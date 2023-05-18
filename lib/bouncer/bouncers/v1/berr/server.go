package berr

import (
	"fmt"

	"pggat2/lib/perror"
)

type Server struct {
	Error error
}

func (Server) IsServer() bool {
	return true
}

func (Server) IsClient() bool {
	return false
}

func (T Server) PError() perror.Error {
	return perror.New(perror.ERROR, perror.InternalError, fmt.Sprintf("server error: %s", T.Error.Error()))
}

func (T Server) String() string {
	return T.Error.Error()
}

func (Server) err() {}

var _ Error = Server{}
