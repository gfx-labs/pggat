package backends

import (
	"pggat2/lib/auth"
	"pggat2/lib/bouncer"
	"pggat2/lib/util/strutil"
)

type AcceptOptions struct {
	SSLMode           bouncer.SSLMode
	Credentials       auth.Credentials
	Database          string
	StartupParameters map[strutil.CIString]string
}
