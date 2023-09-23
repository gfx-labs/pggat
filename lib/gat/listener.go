package gat

import (
	"net"

	"pggat/lib/bouncer/frontends/v0"
	"pggat/lib/fed"
)

type FrontendAcceptOptions = frontends.AcceptOptions

type Listener struct {
	Listener net.Listener
	Options  FrontendAcceptOptions
}

func (T Listener) Accept() (fed.Conn, error) {
	raw, err := T.Listener.Accept()
	if err != nil {
		return nil, err
	}
	conn := fed.WrapNetConn(raw)
	_, err = frontends.Accept(&frontends.AcceptContext{
		Conn:    conn,
		Options: T.Options,
	})
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (T Listener) Close() error {
	return T.Listener.Close()
}
