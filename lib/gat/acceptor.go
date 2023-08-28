package gat

import (
	"net"

	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/zap"
)

type Acceptor struct {
	Listener net.Listener
	Options  frontends.AcceptOptions
}

func (T Acceptor) Accept() (zap.Conn, frontends.AcceptParams, error) {
	netConn, err := T.Listener.Accept()
	if err != nil {
		return nil, frontends.AcceptParams{}, err
	}
	conn := zap.WrapNetConn(netConn)
	params, err := frontends.Accept(conn, T.Options)
	if err != nil {
		_ = conn.Close()
		return nil, frontends.AcceptParams{}, err
	}
	return conn, params, nil
}

func Listen(network, address string, options frontends.AcceptOptions) (Acceptor, error) {
	listener, err := net.Listen(network, address)
	if err != nil {
		return Acceptor{}, err
	}
	return Acceptor{
		Listener: listener,
		Options:  options,
	}, nil
}

func Serve(acceptor Acceptor, gat *Gat) error {
	for {
		conn, params, err := acceptor.Accept()
		if err != nil {
			continue
		}
		go func() {
			_ = gat.Serve(conn, params)
		}()
	}
}

func ListenAndServe(network, address string, options frontends.AcceptOptions, gat *Gat) error {
	listener, err := Listen(network, address, options)
	if err != nil {
		return err
	}
	return Serve(listener, gat)
}
