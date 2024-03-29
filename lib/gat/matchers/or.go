package matchers

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*Or)(nil))
}

type Or struct {
	Or []json.RawMessage `json:"or" caddy:"namespace=pggat.matchers inline_key=matcher"`

	or []gat.Matcher
}

func (T *Or) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.or",
		New: func() caddy.Module {
			return new(Or)
		},
	}
}

func (T *Or) Provision(ctx caddy.Context) error {
	T.or = make([]gat.Matcher, 0, len(T.Or))
	if T.Or != nil {
		raw, err := ctx.LoadModule(T, "Or")
		if err != nil {
			return fmt.Errorf("loading matcher module: %v", err)
		}

		val := raw.([]any)
		T.or = make([]gat.Matcher, 0, len(val))
		for _, vv := range val {
			T.or = append(T.or, vv.(gat.Matcher))
		}
	}

	return nil
}

func (T *Or) Matches(conn *fed.Conn) bool {
	for _, matcher := range T.or {
		if matcher.Matches(conn) {
			return true
		}
	}
	return false
}

var _ gat.Matcher = (*Or)(nil)
var _ caddy.Module = (*Or)(nil)
var _ caddy.Provisioner = (*Or)(nil)
