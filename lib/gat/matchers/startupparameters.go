package matchers

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	caddy.RegisterModule((*StartupParameters)(nil))
}

type StartupParameters struct {
	Parameters map[string]string `json:"startup_parameters"`

	parameters map[strutil.CIString]string
}

func (T *StartupParameters) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.startup_parameters",
		New: func() caddy.Module {
			return new(StartupParameters)
		},
	}
}

func (T *StartupParameters) Provision(ctx caddy.Context) error {
	T.parameters = make(map[strutil.CIString]string, len(T.Parameters))
	for key, value := range T.Parameters {
		T.parameters[strutil.MakeCIString(key)] = value
	}

	return nil
}

func (T *StartupParameters) Matches(conn *fed.Conn) bool {
	for key, value := range T.parameters {
		if conn.InitialParameters[key] != value {
			return false
		}
	}
	return true
}

var _ gat.Matcher = (*StartupParameters)(nil)
var _ caddy.Module = (*StartupParameters)(nil)
var _ caddy.Provisioner = (*StartupParameters)(nil)
