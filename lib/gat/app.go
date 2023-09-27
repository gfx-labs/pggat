package gat

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/util/dur"
	"gfx.cafe/gfx/pggat/lib/util/maps"
)

type Config struct {
	StatLogPeriod dur.Duration     `json:"stat_log_period"`
	Listen        []ListenerConfig `json:"listen"`
	Servers       []ServerConfig   `json:"servers"`
}

func init() {
	caddy.RegisterModule((*App)(nil))
}

type App struct {
	Config

	listen  []*Listener
	servers []*Server

	keys maps.RWLocked[[8]byte, *Pool]
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

func (T *App) Start() error {
	// start listeners
	for _, listener := range T.listen {
		if err := listener.Start(); err != nil {
			return err
		}
	}

	return nil
}

func (T *App) Stop() error {
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
