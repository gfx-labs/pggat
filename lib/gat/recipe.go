package gat

import (
	"net"

	"pggat2/lib/auth"
	"pggat2/lib/bouncer"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
)

type Recipe interface {
	Dial() (zap.ReadWriter, error)
	Connect() (bouncer.Conn, error)

	GetMinConnections() int
	// GetMaxConnections returns the maximum amount of connections for this db
	// Return 0 for unlimited connections
	GetMaxConnections() int
}

type TCPRecipe struct {
	Database    string
	Address     string
	Credentials auth.Credentials

	MinConnections int
	MaxConnections int

	StartupParameters map[strutil.CIString]string
}

func (T TCPRecipe) Dial() (zap.ReadWriter, error) {
	conn, err := net.Dial("tcp", T.Address)
	if err != nil {
		return nil, err
	}
	rw := zap.WrapNetConn(conn)
	return rw, nil
}

func (T TCPRecipe) Connect() (bouncer.Conn, error) {
	rw, err := T.Dial()
	if err != nil {
		return bouncer.Conn{}, err
	}

	server, err := backends.Accept(rw, backends.AcceptOptions{
		Credentials:       T.Credentials,
		Database:          T.Database,
		StartupParameters: T.StartupParameters,
	})
	if err != nil {
		return bouncer.Conn{}, err
	}

	return server, nil
}

func (T TCPRecipe) GetMinConnections() int {
	return T.MinConnections
}

func (T TCPRecipe) GetMaxConnections() int {
	return T.MaxConnections
}
