package caddy

import "github.com/caddyserver/caddy/v2"

func init() {
	caddy.RegisterModule(Postgres{})
}

type Postgres struct{}

func (T Postgres) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "postgres",
		New: func() caddy.Module {
			return Postgres{}
		},
	}
}

func (T Postgres) Start() error {
	// TODO(garet)
	return nil
}

func (T Postgres) Stop() error {
	// TODO(garet)
	return nil
}

var _ caddy.Module = Postgres{}
var _ caddy.App = Postgres{}
