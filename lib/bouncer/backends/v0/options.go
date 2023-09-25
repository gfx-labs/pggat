package backends

import (
	"crypto/tls"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type AcceptOptions struct {
	SSLMode           bouncer.SSLMode
	SSLConfig         *tls.Config
	Username          string
	Credentials       auth.Credentials
	Database          string
	StartupParameters map[strutil.CIString]string
}
