package ps

import (
	"errors"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/middleware"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Server struct {
	parameters map[strutil.CIString]string
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

func (T *Server) Write(_ middleware.Context, _ fed.Packet) error {
	return nil
}

var _ middleware.Middleware = (*Server)(nil)
