package eqp

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
)

type Server struct {
	state State
}

func NewServer() *Server {
	return new(Server)
}

func (T *Server) PreRead(ctx context.Context, _ bool) (fed.Packet, error) {
	return nil, nil
}

func (T *Server) ReadPacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	return T.state.S2C(packet)
}

func (T *Server) WritePacket(ctx context.Context, packet fed.Packet) (fed.Packet, error) {
	return T.state.C2S(packet)
}

func (T *Server) PostWrite(ctx context.Context, ) (fed.Packet, error) {
	return nil, nil
}

var _ fed.Middleware = (*Server)(nil)
