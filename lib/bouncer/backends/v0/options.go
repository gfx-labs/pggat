package backends

import (
	"pggat2/lib/auth"
	"pggat2/lib/util/strutil"
)

type AcceptOptions struct {
	Credentials       auth.Credentials
	Database          string
	StartupParameters map[strutil.CIString]string
}
