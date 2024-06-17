package listeners

import (
	"crypto/tls"
	"strings"

	"gfx.cafe/gfx/pggat/lib/fed"
	"github.com/caddyserver/caddy/v2"
)

const Namespace = "pggat.listeners"

func WithNamespace(suffix ...string) string {
	if len(suffix) == 0 || suffix[0] == "" {
		return Namespace
	}
	return Namespace + "." + strings.Join(suffix, ".")
}

type Listener interface {
	caddy.App
	caddy.Provisioner
	caddy.Module

	TLSConfig() (bool, *tls.Config)
	Accept() (*fed.Conn, error)
}

type SSLServer interface {
	ServerTLSConfig() *tls.Config
}
