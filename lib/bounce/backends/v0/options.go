package backends

import (
	"crypto/tls"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type acceptOptions struct {
	SSLMode           bounce.SSLMode
	SSLConfig         *tls.Config
	Username          string
	Credentials       auth.Credentials
	Database          string
	StartupParameters map[strutil.CIString]string
}
