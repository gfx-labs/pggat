package ps

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Server struct {
	parameters map[string]string

	middleware.Nil
}

func MakeServer() Server {
	return Server{
		parameters: make(map[string]string),
	}
}

func (T *Server) Read(_ middleware.Context, in zap.In) error {
	switch in.Type() {
	case packets.ParameterStatus:
		key, value, ok := packets.ReadParameterStatus(in)
		if !ok {
			return errors.New("bad packet format")
		}
		T.parameters[key] = value
	}
	return nil
}

var _ middleware.Middleware = (*Server)(nil)
