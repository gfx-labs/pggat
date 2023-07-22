package gat

import "reflect"

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

func RecipesEqual(a, b Recipe) bool {
	return reflect.DeepEqual(a, b)
}
