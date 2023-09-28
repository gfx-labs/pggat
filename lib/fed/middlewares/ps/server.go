package ps

import (
	"errors"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
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

func (T *Server) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	switch packet.Type() {
	case packets.TypeParameterStatus:
		var ps packets.ParameterStatus
		if !ps.ReadFromPacket(packet) {
			return packet, errors.New("bad packet format j")
		}
		ikey := strutil.MakeCIString(ps.Key)
		if T.parameters == nil {
			T.parameters = make(map[strutil.CIString]string)
		}
		T.parameters[ikey] = ps.Value
	}
	return packet, nil
}

func (T *Server) WritePacket(packet fed.Packet) (fed.Packet, error) {
	return packet, nil
}

var _ fed.Middleware = (*Server)(nil)
