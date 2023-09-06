package recipe

import (
	"pggat2/lib/gat/pool/recipe/dialer"
)

type Options struct {
	Dialer dialer.Dialer

	MinConnections int
	// MaxConnections is the max number of simultaneous connections from this recipe. 0 = unlimited
	MaxConnections int
}
