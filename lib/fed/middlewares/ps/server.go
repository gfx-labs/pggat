package ps

import (
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

func (T *Server) PreRead(_ bool) (fed.Packet, error) {
	return nil, nil
}

func (T *Server) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	switch packet.Type() {
	case packets.TypeParameterStatus:
		var p packets.ParameterStatus
		err := fed.ToConcrete(&p, packet)
		if err != nil {
			return nil, err
		}
		ikey := strutil.MakeCIString(p.Key)
		if T.parameters == nil {
			T.parameters = make(map[strutil.CIString]string)
		}
		T.parameters[ikey] = p.Value
		return &p, nil
	default:
		return packet, nil
	}
}

func (T *Server) WritePacket(packet fed.Packet) (fed.Packet, error) {
	return packet, nil
}

func (T *Server) PostWrite() (fed.Packet, error) {
	return nil, nil
}

var _ fed.Middleware = (*Server)(nil)
