package backends

import "pggat2/lib/backend"

type Backend struct {
}

var _ backend.Backend = (*Backend)(nil)
