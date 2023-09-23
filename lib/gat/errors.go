package gat

import "fmt"

type errPoolNotFound struct {
	User     string
	Database string
}

func (T errPoolNotFound) Error() string {
	return fmt.Sprintf("pool not found: user=%s database=%s", T.User, T.Database)
}

var _ error = errPoolNotFound{}
