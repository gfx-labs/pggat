package backends

import (
	"crypto/tls"

	"pggat/lib/auth"
	"pggat/lib/bouncer"
	"pggat/lib/util/strutil"
)

type AcceptOptions struct {
	SSLMode           bouncer.SSLMode
	SSLConfig         *tls.Config
	Credentials       auth.Credentials
	Database          string
	StartupParameters map[strutil.CIString]string
}
