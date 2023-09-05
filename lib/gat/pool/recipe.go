package pool

import "pggat2/lib/gat/pool/dialer"

type Recipe struct {
	Dialer         dialer.Dialer
	MinConnections int
	// MaxConnections is the max number of active server connections for this recipe.
	// 0 = unlimited
	MaxConnections int
}
