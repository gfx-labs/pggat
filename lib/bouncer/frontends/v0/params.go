package frontends

import "gfx.cafe/gfx/pggat/lib/fed"

type acceptParams struct {
	CancelKey   fed.BackendKey
	IsCanceling bool
}
