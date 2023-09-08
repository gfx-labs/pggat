package dialer

import (
	"net"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/fed"
)

type Net struct {
	Network string
	Address string

	AcceptOptions backends.AcceptOptions
}

func (T Net) Dial() (fed.Conn, backends.AcceptParams, error) {
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

func (T Net) Cancel(key [8]byte) error {
	c, err := net.Dial(T.Network, T.Address)
	if err != nil {
		return err
	}
	conn := fed.WrapNetConn(c)
	defer func() {
		_ = conn.Close()
	}()
	return backends.Cancel(conn, key)
}

var _ Dialer = Net{}