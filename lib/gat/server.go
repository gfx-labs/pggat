package gat

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type ServerConfig struct {
	Match                    json.RawMessage    `json:"match" caddy:"namespace=pggat.matchers inline_key=matcher"`
	AllowedStartupParameters []strutil.CIString `json:"allowed_startup_parameters"`
	Routes                   []RouteConfig      `json:"routes"`
}

type Server struct {
	ServerConfig

	match  Matcher
	routes []*Route
}

func (T *Server) Provision(ctx caddy.Context) error {
	if T.Match != nil {
		val, err := ctx.LoadModule(T, "Match")
		if err != nil {
			return fmt.Errorf("loading matcher module: %v", err)
		}
		T.match = val.(Matcher)
	}

	T.routes = make([]*Route, 0, len(T.Routes))
	for _, config := range T.Routes {
		route := &Route{
			RouteConfig: config,
		}
		if err := route.Provision(ctx); err != nil {
			return err
		}
		T.routes = append(T.routes, route)
	}

	return nil
}

func (T *Server) lookup(conn *fed.Conn) *pool.Pool {
	for _, route := range T.routes {
		if route.match != nil && !route.match.Matches(conn) {
			continue
		}

		p := route.provide.Lookup(conn)
		if p != nil {
			return p
		}
	}

	return nil
}

var _ caddy.Provisioner = (*Server)(nil)
