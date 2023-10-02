package fed

import "crypto/tls"

type SSL interface {
	SSL() bool
}

type SSLClient interface {
	SSL

	EnableSSLClient(config *tls.Config) error
}

type SSLServer interface {
	SSL

	EnableSSLServer(config *tls.Config) error
}
