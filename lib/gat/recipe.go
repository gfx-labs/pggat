package gat

import (
	"net"

	"pggat2/lib/auth"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/util/maps"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
)

type Recipe interface {
	Connect() (zap.ReadWriter, map[strutil.CIString]string, error)

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

func (T TCPRecipe) Connect() (zap.ReadWriter, map[strutil.CIString]string, error) {
	conn, err := net.Dial("tcp", T.Address)
	if err != nil {
		return nil, nil, err
	}
	rw := zap.WrapIOReadWriter(conn)

	parameterStatus := maps.Clone(T.StartupParameters)
	if parameterStatus == nil {
		parameterStatus = make(map[strutil.CIString]string)
	}

	err = backends.Accept(rw, T.Credentials, T.Database, parameterStatus)
	if err != nil {
		return nil, nil, err
	}

	return rw, parameterStatus, nil
}

func (T TCPRecipe) GetMinConnections() int {
	return T.MinConnections
}

func (T TCPRecipe) GetMaxConnections() int {
	return T.MaxConnections
}
