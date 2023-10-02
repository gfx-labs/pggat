package gat

import "crypto/tls"

type SSLServer interface {
	ServerTLSConfig() *tls.Config
}

type SSLClient interface {
	ClientTLSConfig() *tls.Config
}
