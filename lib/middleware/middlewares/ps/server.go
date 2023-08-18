package ps

import (
	"errors"

	"pggat2/lib/middleware"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Server struct {
	parameters map[strutil.CIString]string

	middleware.Nil
}

func NewServer(parameters map[strutil.CIString]string) *Server {
	return &Server{
		parameters: parameters,
	}
}

func (T *Server) Read(_ middleware.Context, in *zap.Packet) error {
	switch in.ReadType() {
	case packets.ParameterStatus:
		key, value, ok := packets.ReadParameterStatus(in.Read())
		if !ok {
			return errors.New("bad packet format")
		}
		ikey := strutil.MakeCIString(key)
		if T.parameters == nil {
			T.parameters = make(map[strutil.CIString]string)
		}
		T.parameters[ikey] = value
	}
	return nil
}

var _ middleware.Middleware = (*Server)(nil)
