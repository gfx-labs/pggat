package frontends

import "pggat2/lib/util/strutil"

type AcceptParams struct {
	CancelKey [8]byte

	// or

	SSLEnabled        bool
	User              string
	Database          string
	InitialParameters map[strutil.CIString]string
}

type AuthenticateParams struct {
	BackendKey [8]byte
}
