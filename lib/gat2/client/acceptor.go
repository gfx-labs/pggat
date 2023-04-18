package client

import "net"

type Acceptor interface {
	Accept(net.Conn) (Client, error)
}

type AcceptorFunc func(net.Conn) (Client, error)

func (T AcceptorFunc) Accept(conn net.Conn) (Client, error) {
	return T(conn)
}
