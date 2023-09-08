package frontends

import (
	"crypto/tls"

	"pggat/lib/auth"
	"pggat/lib/util/strutil"
)

type AcceptOptions struct {
	SSLRequired           bool
	SSLConfig             *tls.Config
	AllowedStartupOptions []strutil.CIString
}

type AuthenticateOptions struct {
	Credentials auth.Credentials
}
