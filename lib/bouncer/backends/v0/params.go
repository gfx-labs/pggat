package backends

import "gfx.cafe/gfx/pggat/lib/util/strutil"

type acceptParams struct {
	SSLEnabled        bool
	InitialParameters map[strutil.CIString]string
	BackendKey        [8]byte
}
