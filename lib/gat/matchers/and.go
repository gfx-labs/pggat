package matchers

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*And)(nil))
}

type And struct {
	And []json.RawMessage `json:"and" caddy:"namespace=pggat.matchers inline_key=matcher"`

	and []gat.Matcher
}

func (T *And) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.and",
		New: func() caddy.Module {
			return new(And)
		},
	}
}

func (T *And) Provision(ctx caddy.Context) error {
	T.and = make([]gat.Matcher, 0, len(T.And))
	if T.And != nil {
		val, err := ctx.LoadModule(T, "And")
		if err != nil {
			return fmt.Errorf("loading matcher module: %v", err)
		}

		for _, vv := range val.([]any) {
			T.and = append(T.and, vv.(gat.Matcher))
		}
	}

	return nil
}

var _ gat.Matcher = (*And)(nil)
var _ caddy.Module = (*And)(nil)
var _ caddy.Provisioner = (*And)(nil)
