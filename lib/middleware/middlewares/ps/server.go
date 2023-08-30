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

func (T *Server) Read(_ middleware.Context, packet zap.Packet) error {
	switch packet.Type() {
	case packets.TypeParameterStatus:
		var ps packets.ParameterStatus
		if !ps.ReadFromPacket(packet) {
			return errors.New("bad packet format j")
		}
		ikey := strutil.MakeCIString(ps.Key)
		if T.parameters == nil {
			T.parameters = make(map[strutil.CIString]string)
		}
		T.parameters[ikey] = ps.Value
	}
	return nil
}

var _ middleware.Middleware = (*Server)(nil)
