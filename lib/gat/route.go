package gat

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"
)

type RouteConfig struct {
	Match   json.RawMessage `json:"match" caddy:"namespace=pggat.matchers inline_key=matcher"`
	Provide json.RawMessage `json:"provide" caddy:"namespace=pggat.providers inline_key=provider"`
}

type Route struct {
	RouteConfig

	match   Matcher
	provide Provider
}

func (T *Route) Provision(ctx caddy.Context) error {
	if T.Match != nil {
		val, err := ctx.LoadModule(T, "Match")
		if err != nil {
			return fmt.Errorf("loading matcher module: %v", err)
		}
		T.match = val.(Matcher)
	}
	if T.Provide != nil {
		val, err := ctx.LoadModule(T, "Provide")
		if err != nil {
			return fmt.Errorf("loading provider module: %v", err)
		}
		T.provide = val.(Provider)
	}
	return nil
}

var _ caddy.Provisioner = (*Route)(nil)
