package caddy

import (
	"encoding/json"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddyconfig.RegisterAdapter("caddyfile", caddyfile.Adapter{ServerType: ServerType{}})
}

type ServerType struct{}

func (ServerType) Setup(blocks []caddyfile.ServerBlock, m map[string]any) (*caddy.Config, []caddyconfig.Warning, error) {
	var config caddy.Config
	var warnings []caddyconfig.Warning

	var postgres PGGat

	for _, block := range blocks {
		var server Server
		for _, segment := range block.Segments {
			if info, ok := gat.GetModule(segment.Directive()); ok {
				server.Modules = append(server.Modules, ServerModule{
					Type: info.ID,
				})
			}
		}

		server.Listen = make([]ServerSlug, 0, len(block.Keys))
		for _, key := range block.Keys {
			var slug ServerSlug
			if err := slug.FromString(key); err != nil {
				return nil, nil, err
			}

			server.Listen = append(server.Listen, slug)
		}

		postgres.Servers = append(postgres.Servers, server)
	}

	if config.AppsRaw == nil {
		config.AppsRaw = make(caddy.ModuleMap)
	}
	raw, err := json.Marshal(postgres)
	if err != nil {
		return nil, nil, err
	}
	config.AppsRaw["pggat"] = raw

	return &config, warnings, nil
}

var _ caddyfile.ServerType = ServerType{}
