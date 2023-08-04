package gat

import (
	"net"

	"pggat2/lib/zap"
)

type Recipe interface {
	Connect() (zap.ReadWriter, error)

	GetDatabase() string
	GetUser() string
	GetPassword() string

	GetStartupParameters() map[string]string

	GetMinConnections() int
	GetMaxConnections() int
}

type TCPRecipe struct {
	Database string
	Address  string
	User     string
	Password string

	MinConnections int
	MaxConnections int
}

func (T TCPRecipe) Connect() (zap.ReadWriter, error) {
	conn, err := net.Dial("tcp", T.Address)
	if err != nil {
		return nil, err
	}
	rw := zap.WrapIOReadWriter(conn)
	return rw, nil
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
	return nil
}

func (T TCPRecipe) GetMinConnections() int {
	return T.MinConnections
}

func (T TCPRecipe) GetMaxConnections() int {
	return T.MaxConnections
}
