package gsql

import "net"

type Addr struct{}

func (Addr) Network() string {
	return "gsql"
}

func (Addr) String() string {
	return "local gsql client"
}

var _ net.Addr = Addr{}
