package insecure_skip_verify

import (
	"crypto/tls"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*Client)(nil))
}

type Client struct {
}

func (T *Client) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.ssl.clients.insecure_skip_verify",
		New: func() caddy.Module {
			return new(Client)
		},
	}
}

func (T *Client) ClientTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
	}
}

var _ gat.SSLClient = (*Client)(nil)
var _ caddy.Module = (*Client)(nil)
