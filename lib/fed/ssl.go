package fed

import "crypto/tls"

type SSLClient interface {
	EnableSSLClient(config *tls.Config) error
}

type SSLServer interface {
	EnableSSLServer(config *tls.Config) error
}
