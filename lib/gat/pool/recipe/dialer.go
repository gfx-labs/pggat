package recipe

import (
	"errors"
	"io"
	"net"

	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
)

type BackendAcceptOptions = backends.AcceptOptions

type Dialer struct {
	Network string
	Address string

	AcceptOptions BackendAcceptOptions
}

func (T Dialer) Dial() (fed.Conn, backends.AcceptParams, error) {
	c, err := net.Dial(T.Network, T.Address)
	if err != nil {
		return nil, backends.AcceptParams{}, err
	}
	conn := fed.WrapNetConn(c)
	ctx := backends.AcceptContext{
		Conn:    conn,
		Options: T.AcceptOptions,
	}
	params, err := backends.Accept(&ctx)
	if err != nil {
		return nil, backends.AcceptParams{}, err
	}
	return conn, params, nil
}

func (T Dialer) Cancel(key [8]byte) error {
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
	_, err = conn.ReadPacket(true, nil)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}
