package gnetcodec

import (
	"context"
	"log/slog"
	"net"
	"sync/atomic"

	"github.com/caddyserver/caddy/v2"
	"github.com/panjf2000/gnet/v2"
)

type Server struct {
	opts []gnet.Option

	log *slog.Logger

	ready  chan struct{}
	conns  chan *Codec
	closed chan error

	gnet.BuiltinEventEngine
	eng       gnet.Engine
	connected atomic.Int64
}

func (s *Server) Accept() (*Codec, error) {
	val, ok := <-s.conns
	if ok {
		return val, nil
	}
	// TODO: maybe use a better error here :)
	return nil, net.ErrClosed
}

func (s *Server) StartServer(ctx context.Context, addr string, opts ...gnet.Option) error {
	go func() {
		err := gnet.Run(s, addr, opts...)
		if err != nil {
			s.closed <- err
		}
		close(s.closed)
	}()
	select {
	case err := <-s.closed:
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
	T.closed = make(chan error)
	T.conns = make(chan *Codec)
	return nil
}

func (s *Server) OnOpen(c gnet.Conn) ([]byte, gnet.Action) {
	slog.Info("new conn", "conn", c.RemoteAddr().String())
	dec := NewCodec()
	c.SetContext(dec)
	dec.conn = c
	dec.encoder.Reset(c)
	s.connected.Add(1)
	// TODO: consider an ant pool, but likely not needed here
	// if we fall behind on accept handling, that seems to be our own fault i think.
	go func() {
		s.conns <- dec
	}()
	return nil, gnet.None
}

func (s *Server) OnTraffic(c gnet.Conn) gnet.Action {
	//slog.Info("got traffic", "conn", c.RemoteAddr().String())
	uc := c.Context().(*Codec)
	uc.mu.Lock()
	defer uc.mu.Lock()
	bts, err := c.Next(-1)
	if err != nil {
		s.log.Error("short read", "conn", c.RemoteAddr(), "err", err)
		return gnet.Close
	}
	if len(bts) > 0 {
		err = uc.OnData(bts)
		if err != nil {
			s.log.Error("data pipe", "conn", c.RemoteAddr(), "err", err)
			return gnet.Close
		}
	}
	return gnet.None
}

func (s *Server) OnBoot(eng gnet.Engine) gnet.Action {
	close(s.ready)
	return gnet.None
}
func (s *Server) OnShutdown(eng gnet.Engine) {
	close(s.conns)
}

func (s *Server) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	if err != nil {
		s.log.Warn("conn fatal error", "conn", c.RemoteAddr().String(), "err", err)
	}
	s.connected.Add(-1)
	slog.Warn("conn disconnected", "conn", c.RemoteAddr().String())

	return gnet.None
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.eng.Stop(ctx)
}
