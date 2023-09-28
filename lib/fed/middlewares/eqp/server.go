package eqp

import (
	"gfx.cafe/gfx/pggat/lib/fed"
)

type Server struct {
	state State
}

func NewServer() *Server {
	return new(Server)
}

func (T *Server) ReadPacket(packet fed.Packet) (fed.Packet, error) {
	T.state.S2C(packet)
	return packet, nil
}

func (T *Server) WritePacket(packet fed.Packet) (fed.Packet, error) {
	T.state.C2S(packet)
	return packet, nil
}

var _ fed.Middleware = (*Server)(nil)
