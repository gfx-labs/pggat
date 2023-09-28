package gat

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"
)

type RouteConfig struct {
	Match  json.RawMessage `json:"match" caddy:"namespace=pggat.matchers inline_key=matcher"`
	Handle json.RawMessage `json:"handle" caddy:"namespace=pggat.handlers inline_key=handler"`
}

type Route struct {
	RouteConfig

	match  Matcher
	handle Handler
}

func (T *Route) Provision(ctx caddy.Context) error {
	if T.Match != nil {
		val, err := ctx.LoadModule(T, "Match")
		if err != nil {
			return fmt.Errorf("loading matcher module: %v", err)
		}
		T.match = val.(Matcher)
	}
	if T.Handle != nil {
		val, err := ctx.LoadModule(T, "Handle")
		if err != nil {
			return fmt.Errorf("loading handle module: %v", err)
		}
		T.handle = val.(Handler)
	}
	return nil
}

var _ caddy.Provisioner = (*Route)(nil)
