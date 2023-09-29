package gatcaddyfile

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/ssl/clients/insecure_skip_verify"
	"gfx.cafe/gfx/pggat/lib/gat/ssl/servers/self_signed"
)

func init() {
	RegisterDirective(SSLServer, "self_signed", func(_ *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		return &self_signed.Server{}, nil
	})

	RegisterDirective(SSLClient, "insecure_skip_verify", func(_ *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		return &insecure_skip_verify.Client{}, nil
	})
}
