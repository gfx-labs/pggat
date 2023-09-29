package gatcaddyfile

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/ssl/servers/self_signed"
)

var sslServers map[string]Unmarshaller

func RegisterSSLServerDirective(directive string, unmarshaller Unmarshaller) {
	if _, ok := sslServers[directive]; ok {
		panic(fmt.Sprintf(`duplicate ssl server directive "%s"`, directive))
	}
	if sslServers == nil {
		sslServers = make(map[string]Unmarshaller)
	}
	sslServers[directive] = unmarshaller
}

func init() {
	RegisterSSLServerDirective("self_signed", func(_ *caddyfile.Dispenser) (caddy.Module, error) {
		return &self_signed.Server{}, nil
	})
}
