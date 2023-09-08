package ps

import (
	"errors"

	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/middleware"
	"pggat/lib/util/strutil"
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

func (T *Server) Read(_ middleware.Context, packet fed.Packet) error {
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
