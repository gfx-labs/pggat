package recipe

type Config struct {
	Dialer Dialer

	MinConnections int
	// MaxConnections is the max number of active server connections for this recipe.
	// 0 = unlimited
	MaxConnections int
}
