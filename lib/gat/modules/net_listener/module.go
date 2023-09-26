package net_listener

import (
	"errors"
	"net"

	"tuxpa.in/a/zlog/log"

	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
)

type Module struct {
	Config

	listener net.Listener
	accepted chan gat.AcceptedConn
}

func (*Module) GatModule() {}

func (T *Module) Start() error {
	if T.listener != nil {
		// in case this listener was started early
		return nil
	}

	var err error
	T.listener, err = net.Listen(T.Network, T.Address)
	if err != nil {
		return err
	}
	log.Printf("listening on %v", T.listener.Addr())

	T.accepted = make(chan gat.AcceptedConn)
	go T.acceptLoop()

	return nil
}

func (T *Module) Stop() error {
	return T.listener.Close()
}

func (T *Module) Addr() net.Addr {
	if T.listener == nil {
		return nil
	}
	return T.listener.Addr()
}

func (T *Module) accept(raw net.Conn) {
	conn := fed.WrapNetConn(raw)
	ctx := frontends.AcceptContext{
		Conn:    conn,
		Options: T.AcceptOptions,
	}
	params, err := frontends.Accept(&ctx)
	if err != nil {
		log.Printf("failed to accept conn: %v", err)
		return
	}
	_ = params // TODO(garet)
	T.accepted <- gat.AcceptedConn{
		Conn:   conn,
		Params: params,
	}
}

func (T *Module) acceptLoop() {
	for {
		conn, err := T.listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return
			}
			log.Printf("failed to accept conn: %v", err)
			continue
		}

		T.accept(conn)
	}
}

func (T *Module) Accept() []<-chan gat.AcceptedConn {
	return []<-chan gat.AcceptedConn{
		T.accepted,
	}
}

var _ gat.Module = (*Module)(nil)
var _ gat.Listener = (*Module)(nil)
var _ gat.Starter = (*Module)(nil)
var _ gat.Stopper = (*Module)(nil)
