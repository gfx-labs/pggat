package pool

import (
	"net"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/fed"
)

type Dialer interface {
	Dial() (fed.Conn, backends.AcceptParams, error)
	Cancel(cancelKey [8]byte) error
}

type NetDialer struct {
	Network string
	Address string

	AcceptOptions backends.AcceptOptions
}

func (T NetDialer) Dial() (fed.Conn, backends.AcceptParams, error) {
	c, err := net.Dial(T.Network, T.Address)
	if err != nil {
		return nil, backends.AcceptParams{}, err
	}
	conn := fed.WrapNetConn(c)
	params, err := backends.Accept(conn, T.AcceptOptions)
	if err != nil {
		return nil, backends.AcceptParams{}, err
	}

	return conn, params, nil
}

func (T NetDialer) Cancel(cancelKey [8]byte) error {
	c, err := net.Dial(T.Network, T.Address)
	if err != nil {
		return err
	}
	conn := fed.WrapNetConn(c)
	defer func() {
		_ = conn.Close()
	}()
	return backends.Cancel(conn, cancelKey)
}
