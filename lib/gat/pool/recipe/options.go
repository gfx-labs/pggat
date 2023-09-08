package recipe

import "pggat/lib/gat/pool/dialer"

type Options struct {
	Dialer dialer.Dialer

	MinConnections int
	// MaxConnections is the max number of active server connections for this recipe.
	// 0 = unlimited
	MaxConnections int
}
