package pool

type Recipe struct {
	Dialer         Dialer
	MinConnections int
	MaxConnections int
}
