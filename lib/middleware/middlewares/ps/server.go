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

func NewServer() *Server {
	return &Server{
		parameters: make(map[string]string),
	}
}

func (T *Server) Read(_ middleware.Context, in *zap.Packet) error {
	read := in.Read()
	switch read.ReadType() {
	case packets.ParameterStatus:
		key, value, ok := packets.ReadParameterStatus(&read)
		if !ok {
			return errors.New("bad packet format")
		}
		T.parameters[key] = value
	}
	return nil
}

var _ middleware.Middleware = (*Server)(nil)
