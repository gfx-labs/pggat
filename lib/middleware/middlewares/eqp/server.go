package eqp

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/middleware"
)

type Server struct {
	state State
}

func NewServer() *Server {
	return new(Server)
}

func (T *Server) Read(_ middleware.Context, packet fed.Packet) error {
	T.state.S2C(packet)
	return nil
}

func (T *Server) Write(_ middleware.Context, packet fed.Packet) error {
	T.state.C2S(packet)
	return nil
}

var _ middleware.Middleware = (*Server)(nil)
