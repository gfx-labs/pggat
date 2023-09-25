package frontends

import (
	"crypto/tls"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type AcceptOptions struct {
	SSLRequired           bool
	SSLConfig             *tls.Config
	AllowedStartupOptions []strutil.CIString
}

type AuthenticateOptions struct {
	Credentials auth.Credentials
}
