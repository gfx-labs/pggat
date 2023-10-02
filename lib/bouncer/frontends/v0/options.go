package frontends

import (
	"crypto/tls"

	"gfx.cafe/gfx/pggat/lib/auth"
)

type acceptOptions struct {
	SSLConfig *tls.Config
}

type authenticateOptions struct {
	Credentials auth.Credentials
}
