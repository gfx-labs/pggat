package gat

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/perror"
)

type ServerConfig struct {
	Listen []ListenerConfig `json:"listen,omitempty"`
	Routes []RouteConfig    `json:"routes,omitempty"`
}

type Server struct {
	ServerConfig

	listen              []*Listener
	routes              []*Route
	cancellableHandlers []CancellableHandler
	metricsHandlers     []MetricsHandler

	log *zap.Logger
}

func (T *Server) Provision(ctx caddy.Context) error {
	T.log = ctx.Logger()

	T.listen = make([]*Listener, 0, len(T.Listen))
	for _, config := range T.Listen {
		listener := &Listener{
			ListenerConfig: config,
		}
		if err := listener.Provision(ctx); err != nil {
			return err
		}
		T.listen = append(T.listen, listener)
	}

	T.routes = make([]*Route, 0, len(T.Routes))
	for _, config := range T.Routes {
		route := &Route{
			RouteConfig: config,
		}
		if err := route.Provision(ctx); err != nil {
			return err
		}
		if cancellableHandler, ok := route.handle.(CancellableHandler); ok {
			T.cancellableHandlers = append(T.cancellableHandlers, cancellableHandler)
		}
		if metricsHandler, ok := route.handle.(MetricsHandler); ok {
			T.metricsHandlers = append(T.metricsHandlers, metricsHandler)
		}
		T.routes = append(T.routes, route)
	}

	return nil
}

func (T *Server) Start() error {
	for _, listener := range T.listen {
		if err := listener.Start(); err != nil {
			return err
		}

		go func(listener *Listener) {
			for {
				if !T.acceptFrom(listener) {
					break
				}
			}
		}(listener)
	}

	return nil
}

func (T *Server) Stop() error {
	for _, listen := range T.listen {
		if err := listen.Stop(); err != nil {
			return err
		}
	}

	return nil
}

func (T *Server) Cancel(key fed.BackendKey) {
	for _, cancellableHandler := range T.cancellableHandlers {
		cancellableHandler.Cancel(key)
	}
}

func (T *Server) ReadMetrics(m *metrics.Server) {
	for _, metricsHandler := range T.metricsHandlers {
		metricsHandler.ReadMetrics(&m.Handler)
	}
}

func (T *Server) Serve(conn *fed.Conn) {
	for _, route := range T.routes {
		if route.match != nil && !route.match.Matches(conn) {
			continue
		}

		if route.handle == nil {
			continue
		}
		err := route.handle.Handle(conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				// normal closure
				return
			}

			errResp := perror.ToPacket(perror.Wrap(err))
			_ = conn.WritePacket(errResp)
			return
		}
	}

	// database not found
	errResp := perror.ToPacket(
		perror.New(
			perror.FATAL,
			perror.InvalidPassword,
			fmt.Sprintf(`Database "%s" not found`, conn.Database),
		),
	)
	_ = conn.WritePacket(errResp)
	T.log.Warn("database not found", zap.String("user", conn.User), zap.String("database", conn.Database))
}

func (T *Server) accept(listener *Listener, conn *fed.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	var tlsConfig *tls.Config
	if listener.ssl != nil {
		tlsConfig = listener.ssl.ServerTLSConfig()
	}

	var cancelKey fed.BackendKey
	var isCanceling bool
	var err error
	cancelKey, isCanceling, err = frontends.Accept(conn, tlsConfig)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			T.log.Warn("error accepting client", zap.Error(err))
		}
		return
	}

	if isCanceling {
		T.Cancel(cancelKey)
		return
	}

	count := listener.open.Add(1)
	defer listener.open.Add(-1)

	if listener.MaxConnections != 0 && int(count) > listener.MaxConnections {
		_ = conn.WritePacket(
			perror.ToPacket(perror.New(
				perror.FATAL,
				perror.TooManyConnections,
				"Too many connections, sorry",
			)),
		)
		return
	}

	T.Serve(conn)
}

func (T *Server) acceptFrom(listener *Listener) bool {
	conn, err := listener.accept()
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			return false
		}
		if netErr, ok := err.(*net.OpError); ok {
			// why can't they just expose this error
			if netErr.Err.Error() == "listener 'closed' 😉" {
				return false
			}
		}
		T.log.Warn("error accepting client", zap.Error(err))
		return true
	}

	go T.accept(listener, conn)
	return true
}

var _ caddy.Provisioner = (*Server)(nil)
