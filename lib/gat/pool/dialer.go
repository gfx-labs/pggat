package pool

import (
	"net"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/zap"
)

type Dialer interface {
	Dial() (zap.Conn, backends.AcceptParams, error)
}

type NetDialer struct {
	Network string
	Address string

	AcceptOptions backends.AcceptOptions
}

func (T NetDialer) Dial() (zap.Conn, backends.AcceptParams, error) {
	c, err := net.Dial(T.Network, T.Address)
	if err != nil {
		return nil, backends.AcceptParams{}, err
	}
	conn := zap.WrapNetConn(c)
	params, err := backends.Accept(conn, T.AcceptOptions)
	if err != nil {
		return nil, backends.AcceptParams{}, err
	}

	return conn, params, nil
}
