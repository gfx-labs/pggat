package matchers

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	caddy.RegisterModule((*Database)(nil))
}

type Database struct {
	Database strutil.Matcher `json:"database"`
}

func (T *Database) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.database",
		New: func() caddy.Module {
			return new(Database)
		},
	}
}

func (T *Database) Matches(conn fed.Conn) bool {
	return T.Database.Matches(conn.Database())
}

var _ gat.Matcher = (*Database)(nil)
var _ caddy.Module = (*Database)(nil)
