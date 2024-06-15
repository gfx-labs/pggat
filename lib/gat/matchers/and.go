package matchers

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
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
		raw, err := ctx.LoadModule(T, "And")
		if err != nil {
			return fmt.Errorf("loading matcher module: %v", err)
		}

		val := raw.([]any)
		T.and = make([]gat.Matcher, 0, len(val))
		for _, vv := range val {
			T.and = append(T.and, vv.(gat.Matcher))
		}
	}

	return nil
}

func (T *And) Matches(conn fed.Conn) bool {
	for _, matcher := range T.and {
		if !matcher.Matches(conn) {
			return false
		}
	}
	return true
}

var _ gat.Matcher = (*And)(nil)
var _ caddy.Module = (*And)(nil)
var _ caddy.Provisioner = (*And)(nil)
