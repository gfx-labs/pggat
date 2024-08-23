package gat

import (
	"context"
	"gfx.cafe/util/go/gotel"
	"time"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/gat/metrics"
)

type Config struct {
	StatLogPeriod caddy.Duration `json:"stat_log_period,omitempty"`
	Servers       []ServerConfig `json:"servers,omitempty"`
}

func init() {
	caddy.RegisterModule((*App)(nil))
}

type App struct {
	Config

	servers []*Server

	closed chan struct{}

	log *zap.Logger

	otelShutdown gotel.ShutdownFunc
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

func (T *App) statLogLoop() {
	t := time.NewTicker(time.Duration(T.StatLogPeriod))
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
	T.otelShutdown, _ = gotel.InitTracing(context.Background(), gotel.WithServiceName("pggat"))

	T.closed = make(chan struct{})
	if T.StatLogPeriod != 0 {
		go T.statLogLoop()
	}

	for _, server := range T.servers {
		if err := server.Start(context.Background()); err != nil {
			return err
		}
	}

	return nil
}

func (T *App) Stop() error {
	defer func() {
		if T.otelShutdown != nil {
			_ = T.otelShutdown(context.Background())
		}
	}()

	close(T.closed)

	for _, server := range T.servers {
		if err := server.Stop(context.Background()); err != nil {
			return err
		}
	}

	return nil
}

var _ caddy.Module = (*App)(nil)
var _ caddy.Provisioner = (*App)(nil)
var _ caddy.App = (*App)(nil)
