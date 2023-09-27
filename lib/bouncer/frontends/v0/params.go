package frontends

import "gfx.cafe/gfx/pggat/lib/util/strutil"

type acceptParams struct {
	CancelKey   [8]byte
	IsCanceling bool

	// or

	SSLEnabled        bool
	User              string
	Database          string
	InitialParameters map[strutil.CIString]string
}

type authenticateParams struct {
	BackendKey [8]byte
}
