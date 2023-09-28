package gat

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/perror"
)

type ServerConfig struct {
	Match  json.RawMessage `json:"match" caddy:"namespace=pggat.matchers inline_key=matcher"`
	Routes []RouteConfig   `json:"routes"`
}

type Server struct {
	ServerConfig

	match               Matcher
	routes              []*Route
	cancellableHandlers []CancellableHandler
	metricsHandlers     []MetricsHandler

	log *zap.Logger
}

func (T *Server) Provision(ctx caddy.Context) error {
	T.log = ctx.Logger()

	if T.Match != nil {
		val, err := ctx.LoadModule(T, "Match")
		if err != nil {
			return fmt.Errorf("loading matcher module: %v", err)
		}
		T.match = val.(Matcher)
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

func (T *Server) Cancel(key [8]byte) {
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

		err := route.handle.Handle(conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				// normal closure
				return
			}

			errResp := packets.ErrorResponse{
				Error: perror.Wrap(err),
			}
			_ = conn.WritePacket(errResp.IntoPacket(nil))
			return
		}
	}

	// database not found
	errResp := packets.ErrorResponse{
		Error: perror.New(
			perror.FATAL,
			perror.InvalidPassword,
			fmt.Sprintf(`Database "%s" not found`, conn.Database),
		),
	}
	_ = conn.WritePacket(errResp.IntoPacket(nil))
	T.log.Warn("database not found", zap.String("user", conn.User), zap.String("database", conn.Database))
}

var _ caddy.Provisioner = (*Server)(nil)
