package x509_key_pair

import (
	"crypto/tls"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/gat"
)

type Server struct {
	CertFile string `json:"cert_file"`
	KeyFile  string `json:"key_file"`

	tlsConfig *tls.Config
}

func (T *Server) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.ssl.servers.x509_key_pair",
		New: func() caddy.Module {
			return new(Server)
		},
	}
}

func (T *Server) Provision(ctx caddy.Context) error {
	cert, err := tls.LoadX509KeyPair(T.CertFile, T.KeyFile)
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
