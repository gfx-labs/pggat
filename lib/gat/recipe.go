package gat

import (
	"net"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/util/maps"
	"pggat2/lib/zap"
)

type Recipe interface {
	Connect() (zap.ReadWriter, map[string]string, error)

	GetMinConnections() int
	// GetMaxConnections returns the maximum amount of connections for this db
	// Return 0 for unlimited connections
	GetMaxConnections() int
}

type TCPRecipe struct {
	Database string
	Address  string
	User     string
	Password string

	MinConnections int
	MaxConnections int

	StartupParameters map[string]string
}

func (T TCPRecipe) Connect() (zap.ReadWriter, map[string]string, error) {
	conn, err := net.Dial("tcp", T.Address)
	if err != nil {
		return nil, nil, err
	}
	rw := zap.WrapIOReadWriter(conn)

	parameterStatus := maps.Clone(T.StartupParameters)

	err = backends.Accept(rw, T.User, T.Password, T.Database, T.StartupParameters)
	if err != nil {
		return nil, nil, err
	}

	return rw, parameterStatus, nil
}

func (T TCPRecipe) GetDatabase() string {
	return T.Database
}

func (T TCPRecipe) GetUser() string {
	return T.User
}

func (T TCPRecipe) GetPassword() string {
	return T.Password
}

func (T TCPRecipe) GetStartupParameters() map[string]string {
	return T.StartupParameters
}

func (T TCPRecipe) GetMinConnections() int {
	return T.MinConnections
}

func (T TCPRecipe) GetMaxConnections() int {
	return T.MaxConnections
}
