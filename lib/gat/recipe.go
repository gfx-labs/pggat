package gat

type Recipe struct {
	// Connection Parameters
	Database string
	Address  string
	User     string
	Password string

	// Config
	MinConnections int
	MaxConnections int
}
