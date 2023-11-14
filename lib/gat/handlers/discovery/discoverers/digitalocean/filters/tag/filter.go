package tag

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/digitalocean/godo"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery/discoverers/digitalocean"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	caddy.RegisterModule((*Filter)(nil))
}

type Filter struct {
	Tag strutil.Matcher `json:"tag"`
}

func (T *Filter) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.discovery.discoverers.digitalocean.filters.tag",
		New: func() caddy.Module {
			return new(Filter)
		},
	}
}

func (T *Filter) Allow(database godo.Database) bool {
	for _, tag := range database.Tags {
		if T.Tag.Matches(tag) {
			return true
		}
	}
	return false
}

func (T *Filter) AllowReplica(database godo.DatabaseReplica) bool {
	for _, tag := range database.Tags {
		if T.Tag.Matches(tag) {
			return true
		}
	}
	return false
}

var _ digitalocean.Filter = (*Filter)(nil)
var _ caddy.Module = (*Filter)(nil)
