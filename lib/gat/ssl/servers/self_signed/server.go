package self_signed

import (
	"crypto/tls"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/util/certs"
)

func init() {
	caddy.RegisterModule((*Server)(nil))
}

type Server struct {
	tlsConfig *tls.Config
}

func (T *Server) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.ssl.servers.self_signed",
		New: func() caddy.Module {
			return new(Server)
		},
	}
}

func (T *Server) Provision(ctx caddy.Context) error {
	cert, err := certs.SelfSign()
	if err != nil {
		return err
	}
	T.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{
			cert,
		},
	}
	return nil
}

func (T *Server) ServerTLSConfig() *tls.Config {
	return T.tlsConfig
}

var _ gat.SSLServer = (*Server)(nil)
var _ caddy.Module = (*Server)(nil)
var _ caddy.Provisioner = (*Server)(nil)
