package matchers

import (
	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*User)(nil))
}

type User struct {
	User string `json:"user"`
}

func (T *User) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.user",
		New: func() caddy.Module {
			return new(User)
		},
	}
}

func (T *User) Matches(conn fed.Conn) bool {
	return conn.User() == T.User
}

var _ gat.Matcher = (*User)(nil)
var _ caddy.Module = (*User)(nil)
