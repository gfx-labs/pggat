package gat

import (
	"crypto/tls"
	"errors"
	"net"
	"time"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/perror"
	"gfx.cafe/gfx/pggat/lib/util/dur"
)

type Config struct {
	StatLogPeriod dur.Duration     `json:"stat_log_period,omitempty"`
	Listen        []ListenerConfig `json:"listen,omitempty"`
	Servers       []ServerConfig   `json:"servers,omitempty"`
}

func init() {
	caddy.RegisterModule((*App)(nil))
}

type App struct {
	Config

	listen  []*Listener
	servers []*Server

	closed chan struct{}

	log *zap.Logger
}

func (T *App) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat",
		New: func() caddy.Module {
			return new(App)
		},
	}
}

func (T *App) Provision(ctx caddy.Context) error {
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

	T.servers = make([]*Server, 0, len(T.Servers))
	for _, config := range T.Servers {
		server := &Server{
			ServerConfig: config,
		}
		if err := server.Provision(ctx); err != nil {
			return err
		}
		T.servers = append(T.servers, server)
	}

	return nil
}

func (T *App) Cancel(key [8]byte) {
	// TODO(garet) make this better (it's probably good enough for now)
	for _, server := range T.servers {
		server.Cancel(key)
	}
}

func (T *App) accept(listener *Listener, conn *fed.Conn) {
	defer func() {
		_ = conn.Close()
	}()

	var tlsConfig *tls.Config
	if listener.ssl != nil {
		tlsConfig = listener.ssl.ServerTLSConfig()
	}

	var cancelKey [8]byte
	var isCanceling bool
	var err error
	cancelKey, isCanceling, err = frontends.Accept(conn, tlsConfig)
	if err != nil {
		T.log.Warn("error accepting client", zap.Error(err))
		return
	}

	if isCanceling {
		T.Cancel(cancelKey)
		return
	}

	for _, server := range T.servers {
		if server.match != nil && !server.match.Matches(conn) {
			continue
		}

		server.Serve(conn)
		return
	}

	T.log.Warn("server not found", zap.String("user", conn.User), zap.String("database", conn.Database))

	errResp := packets.ErrorResponse{
		Error: perror.New(
			perror.FATAL,
			perror.InternalError,
			"No server is available to handle your request",
		),
	}
	_ = conn.WritePacket(errResp.IntoPacket(nil))
}

func (T *App) acceptFrom(listener *Listener) bool {
	conn, err := listener.accept()
	if err != nil {
		if errors.Is(err, net.ErrClosed) {
			return false
		}
		T.log.Warn("error accepting client", zap.Error(err))
		return true
	}

	go T.accept(listener, conn)
	return true
}

func (T *App) statLogLoop() {
	t := time.NewTicker(T.StatLogPeriod.Duration())
	defer t.Stop()

	var stats metrics.Server
	for {
		select {
		case <-t.C:
			for _, server := range T.servers {
				server.ReadMetrics(&stats)
			}
			T.log.Info(stats.String())
			stats.Clear()
		case <-T.closed:
			return
		}
	}
}

func (T *App) Start() error {
	T.closed = make(chan struct{})
	if T.StatLogPeriod != 0 {
		go T.statLogLoop()
	}

	// start listeners
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

func (T *App) Stop() error {
	close(T.closed)

	// stop listeners
	for _, listener := range T.listen {
		if err := listener.Stop(); err != nil {
			return err
		}
	}

	return nil
}

var _ caddy.Module = (*App)(nil)
var _ caddy.Provisioner = (*App)(nil)
var _ caddy.App = (*App)(nil)
