package dialer

import (
	"errors"
	"io"
	"net"

	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/fed"
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
	if err = backends.Cancel(conn, key); err != nil {
		return err
	}

	// wait for server to close the connection, this means that the server received it ok
	_, err = conn.ReadPacket(true)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

var _ Dialer = Net{}
