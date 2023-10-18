package matchers

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	caddy.RegisterModule((*StartupParameter)(nil))
}

type StartupParameter struct {
	Key   strutil.CIString `json:"key"`
	Value strutil.Matcher  `json:"value"`
}

func (T *StartupParameter) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.startup_parameter",
		New: func() caddy.Module {
			return new(StartupParameter)
		},
	}
}

func (T *StartupParameter) Matches(conn *fed.Conn) bool {
	return T.Value.Matches(conn.InitialParameters[T.Key])
}

var _ gat.Matcher = (*StartupParameter)(nil)
var _ caddy.Module = (*StartupParameter)(nil)
