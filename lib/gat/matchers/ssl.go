package matchers

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*SSL)(nil))
}

type SSL struct {
	SSL bool `json:"ssl"`
}

func (T *SSL) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.ssl",
		New: func() caddy.Module {
			return new(SSL)
		},
	}
}

func (T *SSL) Matches(conn *fed.Conn) bool {
	sslConn, ok := conn.ReadWriteCloser.(fed.SSL)
	if !ok {
		return T.SSL == false
	}
	return sslConn.SSL() == T.SSL
}

var _ gat.Matcher = (*SSL)(nil)
var _ caddy.Module = (*SSL)(nil)
