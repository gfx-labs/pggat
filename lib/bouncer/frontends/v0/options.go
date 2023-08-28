package frontends

import (
	"crypto/tls"

	"pggat2/lib/auth"
	"pggat2/lib/util/strutil"
)

type AcceptOptions struct {
	SSLRequired           bool
	SSLConfig             *tls.Config
	AllowedStartupOptions []strutil.CIString
}

type AuthenticateOptions struct {
	Credentials auth.Credentials
}
