package matchers

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*Not)(nil))
}

type Not struct {
	Not json.RawMessage `json:"not" caddy:"namespace=pggat.matchers inline_key=matcher"`

	not gat.Matcher
}

func (T *Not) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.not",
		New: func() caddy.Module {
			return new(Not)
		},
	}
}

func (T *Not) Provision(ctx caddy.Context) error {
	if T.Not != nil {
		val, err := ctx.LoadModule(T, "Not")
		if err != nil {
			return fmt.Errorf("loading matcher module: %v", err)
		}

		T.not = val.(gat.Matcher)
	}

	return nil
}

func (T *Not) Matches(conn fed.Conn) bool {
	return !T.not.Matches(conn)
}

var _ gat.Matcher = (*Not)(nil)
var _ caddy.Module = (*Not)(nil)
var _ caddy.Provisioner = (*Not)(nil)
