package gatcaddyfile

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/ssl/clients/insecure_skip_verify"
	"gfx.cafe/gfx/pggat/lib/gat/ssl/servers/self_signed"
	"gfx.cafe/gfx/pggat/lib/gat/ssl/servers/x509_key_pair"
)

func init() {
	RegisterDirective(SSLServer, "self_signed", func(_ *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		return &self_signed.Server{}, nil
	})
	RegisterDirective(SSLServer, "x509_key_pair", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		var module x509_key_pair.Server

		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		module.CertFile = d.Val()

		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		module.KeyFile = d.Val()

		return &module, nil
	})

	RegisterDirective(SSLClient, "insecure_skip_verify", func(_ *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		return &insecure_skip_verify.Client{}, nil
	})
}
