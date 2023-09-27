package matchers

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"

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
		val, err := ctx.LoadModule(T, "Or")
		if err != nil {
			return fmt.Errorf("loading matcher module: %v", err)
		}

		for _, vv := range val.([]any) {
			T.or = append(T.or, vv.(gat.Matcher))
		}
	}

	return nil
}

var _ gat.Matcher = (*Or)(nil)
var _ caddy.Module = (*Or)(nil)
var _ caddy.Provisioner = (*Or)(nil)
