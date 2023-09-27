package caddy

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*PGGat)(nil))
}

type PGGat struct {
	Servers []Server `json:"servers,omitempty"`

	servers []*gat.Server
}

func (*PGGat) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat",
		New: func() caddy.Module {
			return new(PGGat)
		},
	}
}

func (T *PGGat) Provision(ctx caddy.Context) error {
	T.servers = make([]*gat.Server, 0, len(T.Servers))
	for _, server := range T.Servers {
		var modules []gat.Module
		for _, module := range server.Modules {
			info, ok := gat.GetModule(module.Type)
			if !ok {
				return fmt.Errorf("module not found: %s", module.Type)
			}
			modules = append(modules, info.New())
		}

		T.servers = append(T.servers, gat.NewServer(modules...))
	}

	return nil
}

func (T *PGGat) Start() error {
	for _, server := range T.servers {
		if err := server.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (T *PGGat) Stop() error {
	for _, server := range T.servers {
		if err := server.Stop(); err != nil {
			return err
		}
	}

	return nil
}

var _ caddy.Module = (*PGGat)(nil)
var _ caddy.App = (*PGGat)(nil)
var _ caddy.Provisioner = (*PGGat)(nil)
