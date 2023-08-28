package backends

import "pggat2/lib/util/strutil"

type AcceptParams struct {
	SSLEnabled        bool
	InitialParameters map[strutil.CIString]string
	BackendKey        [8]byte
}
