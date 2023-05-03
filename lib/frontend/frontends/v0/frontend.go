package frontends

import (
	"net"

	"pggat2/lib/frontend"
)

type Frontend struct {
	listener net.Listener
	clients  []*Client
}

func NewFrontend() (*Frontend, error) {
	listener, err := net.Listen("tcp", "0.0.0.0:6432") // TODO(garet) make this configurable
	if err != nil {
		return nil, err
	}
	return &Frontend{
		listener: listener,
	}, nil
}

func (T *Frontend) accept(conn net.Conn) {
	client := NewClient(conn)
	if client != nil {
		T.clients = append(T.clients, client)
	}
}

func (T *Frontend) Run() error {
	for {
		conn, err := T.listener.Accept()
		if err != nil {
			return err
		}

		go T.accept(conn)
	}
}

var _ frontend.Frontend = (*Frontend)(nil)
