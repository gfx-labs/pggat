package backends

import "pggat/lib/util/strutil"

type AcceptParams struct {
	SSLEnabled        bool
	InitialParameters map[strutil.CIString]string
	BackendKey        [8]byte
}
