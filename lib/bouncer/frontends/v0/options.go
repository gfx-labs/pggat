package frontends

import (
	"crypto/tls"

	"pggat2/lib/bouncer"
	"pggat2/lib/util/strutil"
)

type AcceptOptions struct {
	SSLRequired           bool
	SSLConfig             *tls.Config
	Pooler                bouncer.Pooler
	AllowedStartupOptions []strutil.CIString
}
