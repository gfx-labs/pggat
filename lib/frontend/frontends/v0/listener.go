package frontends

import (
	"net"

	"pggat2/lib/frontend"
)

type Listener struct {
	listener net.Listener
}

func NewListener() (*Listener, error) {
	listener, err := net.Listen("tcp", "0.0.0.0:6432") // TODO(garet) make this configurable
	if err != nil {
		return nil, err
	}
	return &Listener{
		listener: listener,
	}, nil
}

func (T *Listener) accept(conn net.Conn) (*Client, error) {
	client := NewClient(conn)
	return client, nil
}

func (T *Listener) Accept() (*Client, error) {
	conn, err := T.listener.Accept()
	if err != nil {
		return nil, err
	}
	return T.accept(conn)
}

func (T *Listener) Listen() error {
	for {
		conn, err := T.listener.Accept()
		if err != nil {
			return err
		}
		go func() {
			NewClient(conn)
		}()
	}
}

var _ frontend.Listener = (*Listener)(nil)
