package gat

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"io"
	"net"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/instrumentation/prom"
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
	tracer              trace.Tracer
	log                 *zap.Logger
}

func (T *Server) Provision(cdyctx caddy.Context) error {
	T.log = cdyctx.Logger()
	T.tracer = otel.Tracer("server", trace.WithInstrumentationAttributes(
		attribute.String("component", "gfx.cafe/gfx/pggat/lib/gat/server.go"),
	))

	// note give Caddy, using the returned context in subsequent provision calls
	//  seems problematic
	_, span := T.tracer.Start(cdyctx.Context, "provision", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	T.listen = make([]*Listener, 0, len(T.Listen))
	for _, config := range T.Listen {
		listener := &Listener{
			ListenerConfig: config,
		}
		if err := listener.Provision(cdyctx); err != nil {
			return err
		}
		T.listen = append(T.listen, listener)
	}

	T.routes = make([]*Route, 0, len(T.Routes))
	for _, config := range T.Routes {
		route := &Route{
			RouteConfig: config,
		}
		if err := route.Provision(cdyctx); err != nil {
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

func (T *Server) Start(ctx context.Context) error {
	ctx, span := T.tracer.Start(ctx, "Start", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	err := T.start(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (T *Server) start(_ context.Context) error {
	for _, listener := range T.listen {
		if err := listener.Start(); err != nil {
			return err
		}

		go func(listener *Listener) {
			for {
				// acceptFrom creates its own context
				if !T.acceptFrom(listener) {
					break
				}
			}
		}(listener)
	}

	return nil
}

func (T *Server) Stop(ctx context.Context) error {
	ctx, span := T.tracer.Start(ctx, "Stop", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	err := T.stop(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (T *Server) stop(_ context.Context) error {
	for _, listen := range T.listen {
		if err := listen.Stop(); err != nil {
			return err
		}
	}

	return nil
}

func (T *Server) Cancel(ctx context.Context, key fed.BackendKey) {
	ctx, span := T.tracer.Start(ctx, "Cancel", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	for _, cancellableHandler := range T.cancellableHandlers {
		cancellableHandler.Cancel(ctx, key)
	}
}

func (T *Server) ReadMetrics(ctx context.Context, m *metrics.Server) {
	ctx, span := T.tracer.Start(ctx, "ReadMetrics", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	for _, metricsHandler := range T.metricsHandlers {
		metricsHandler.ReadMetrics(ctx, &m.Handler)
	}
}

func (T *Server) Serve(ctx context.Context, conn *fed.Conn) {
	ctx, span := T.tracer.Start(ctx, "Serve", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	composed := Router(RouterFunc(func(ctx context.Context, conn *fed.Conn) error {
		// database not found
		errResp := perror.ToPacket(
			perror.New(
				perror.FATAL,
				perror.InvalidPassword,
				fmt.Sprintf(`Database "%s" not found`, conn.Database),
			),
		)
		_ = conn.WritePacket(ctx, errResp)
		T.log.Warn("database not found", zap.String("user", conn.User), zap.String("database", conn.Database))
		return nil
	}))
	for j := len(T.routes) - 1; j >= 0; j-- {
		route := T.routes[j]
		if route.match != nil && !route.match.Matches(conn) {
			continue
		}
		if route.handle == nil {
			continue
		}
		composed = route.handle.Handle(composed)
	}
	err := composed.Route(ctx, conn)
	if err != nil {
		if errors.Is(err, io.EOF) {
			// normal closure
			return
		}

		errResp := perror.ToPacket(perror.Wrap(err))
		_ = conn.WritePacket(ctx, errResp)

		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		return
	}
}

func (T *Server) accept(ctx context.Context, listener *Listener, conn *fed.Conn) {
	defer func() {
		_ = conn.Close(ctx)
	}()
	labels := prom.ListenerLabels{ListenAddr: listener.networkAddress.String()}

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
		T.Cancel(ctx, cancelKey)
		return
	}

	count := listener.open.Add(1)
	prom.Listener.Client(labels).Inc()
	prom.Listener.Incoming(labels).Inc()
	defer func() {
		listener.open.Add(-1)
		prom.Listener.Client(labels).Dec()
	}()

	if listener.MaxConnections != 0 && int(count) > listener.MaxConnections {
		_ = conn.WritePacket(
			ctx,
			perror.ToPacket(perror.New(
				perror.FATAL,
				perror.TooManyConnections,
				"Too many connections, sorry",
			)),
		)
		return
	}
	prom.Listener.Accepted(labels).Inc()
	T.Serve(ctx, conn)
}

func (T *Server) acceptFrom(listener *Listener) bool {
	ctx, span := T.tracer.Start(context.Background(), "acceptFrom", trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	err := listener.listener.Accept(func(c *fed.Conn) {
		T.accept(ctx, listener, c)
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

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
	return true
}

var _ caddy.Provisioner = (*Server)(nil)
