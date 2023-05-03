package backends

import (
	"log"
	"net"

	"pggat2/lib/backend"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
)

type Server struct {
	conn net.Conn

	pnet.Reader
	pnet.Writer
}

func NewServer(conn net.Conn) (*Server, error) {
	server := &Server{
		conn:   conn,
		Reader: pnet.MakeReader(conn),
		Writer: pnet.MakeWriter(conn),
	}
	err := server.accept()
	if err != nil {
		return nil, err
	}
	return server, nil
}

func (T *Server) accept() error {
	var builder packet.Builder
	builder.Int16(3)
	builder.Int16(0)
	builder.String("user")
	builder.String("postgres")
	builder.String("")

	err := T.WriteUntyped(builder.Raw())
	if err != nil {
		return err
	}

	auth, err := T.Read()
	if err != nil {
		return err
	}

	log.Printf("%#v", auth)

	return nil
}

var _ backend.Server = (*Server)(nil)
