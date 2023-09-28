package gatcaddyfile

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func init() {
	caddyconfig.RegisterAdapter("caddyfile", caddyfile.Adapter{ServerType: ServerType{}})
}

type ServerType struct{}

func (ServerType) Setup(blocks []caddyfile.ServerBlock, m map[string]any) (*caddy.Config, []caddyconfig.Warning, error) {
	// TODO implement me
	panic("implement me")
}
