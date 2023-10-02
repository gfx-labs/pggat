package matchers

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*Database)(nil))
}

type Database struct {
	Database string `json:"database"`
}

func (T *Database) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.database",
		New: func() caddy.Module {
			return new(Database)
		},
	}
}

func (T *Database) Matches(conn *fed.Conn) bool {
	return conn.Database == T.Database
}

var _ gat.Matcher = (*Database)(nil)
var _ caddy.Module = (*Database)(nil)
