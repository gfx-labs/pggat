package gfed

import (
	"context"
	"log/slog"
	"sync/atomic"

	"github.com/caddyserver/caddy/v2"
	"github.com/panjf2000/gnet/v2"
)

type Server struct {
	Acceptor func(*Codec)
	opts     []gnet.Option

	log *slog.Logger

	ready chan struct{}
	conns chan *Codec

	gnet.BuiltinEventEngine
	eng       gnet.Engine
	connected atomic.Int64
}

func (s *Server) StartServer(ctx context.Context, addr string, opts ...gnet.Option) error {
	errch := make(chan error)
	go func() {
		err := gnet.Run(s, addr, opts...)
		if err != nil {
			errch <- err
		}
		close(errch)
	}()
	select {
	case err := <-errch:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	case <-s.ready:
		return nil
	}
	return nil
}

func (T *Server) Provision(ctx caddy.Context) error {
	T.log = ctx.Slogger()
	T.ready = make(chan struct{})
	return nil
}

func (s *Server) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	dec := new(Codec)
	c.SetContext(dec)
	dec.localAddr = c.LocalAddr()
	dec.gnetConn = c
	s.connected.Add(1)
	s.Acceptor(dec)
	return nil, gnet.None
}

func (s *Server) OnTraffic(c gnet.Conn) gnet.Action {
	uc := c.Context().(*Codec)
	bts, err := c.Next(c.InboundBuffered())
	if err != nil {
		s.log.Error("short read", "conn", c.RemoteAddr(), "err", err)
		return gnet.Close
	}
	err = uc.OnData(bts)
	if err != nil {
		s.log.Error("data pipe", "conn", c.RemoteAddr(), "err", err)
	}
	return gnet.None
}
func (s *Server) OnBoot(eng gnet.Engine) gnet.Action {
	return gnet.None
}
func (s *Server) OnShutdown(eng gnet.Engine) {
}

func (s *Server) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	if err != nil {
		s.log.Warn("conn fatal error", "conn", c.RemoteAddr().String(), "err", err)
	}
	s.connected.Add(-1)
	slog.Warn("conn disconnected", "conn", c.RemoteAddr().String())

	return gnet.None
}
