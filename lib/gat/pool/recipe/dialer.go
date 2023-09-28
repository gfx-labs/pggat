package recipe

import (
	"crypto/tls"
	"errors"
	"io"
	"net"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Dialer struct {
	Network string
	Address string

	SSLMode           bouncer.SSLMode
	SSLConfig         *tls.Config
	Username          string
	Credentials       auth.Credentials
	Database          string
	StartupParameters map[strutil.CIString]string
}

func (T Dialer) Dial() (*fed.Conn, error) {
	c, err := net.Dial(T.Network, T.Address)
	if err != nil {
		return nil, err
	}
	conn := fed.NewConn(
		fed.NewNetConn(c),
	)
	conn.User = T.Username
	conn.Database = T.Database
	err = backends.Accept(
		conn,
		T.SSLMode,
		T.SSLConfig,
		T.Username,
		T.Credentials,
		T.Database,
		T.StartupParameters,
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (T Dialer) Cancel(key [8]byte) error {
	c, err := net.Dial(T.Network, T.Address)
	if err != nil {
		return err
	}
	conn := fed.NewConn(
		fed.NewNetConn(c),
	)
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
