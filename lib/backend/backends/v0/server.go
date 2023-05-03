package backends

import (
	"errors"
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

	reader := packet.MakeReader(auth)
	switch reader.Type() {
	case packet.Authentication:
		method, ok := reader.Int32()
		if !ok {
			return errors.New("expected authentication method")
		}
		// they have more authentication methods than there are pokemon
		switch method {
		case 0:
			// we're good to go, that was easy
		case 2:
			return errors.New("kerberos v5 is not supported")
		case 3:
			return errors.New("cleartext is not supported")
		case 5:
			return errors.New("md5 password is not supported")
		case 6:
			return errors.New("scm credential is not supported")
		case 7:
			return errors.New("gss is not supported")
		case 9:
			return errors.New("sspi is not supported")
		case 10:
			return errors.New("sasl is not supported")
		default:
			return errors.New("unknown authentication method")
		}
	case packet.ErrorResponse:
		return errors.New("backend errored")
	case packet.NegotiateProtocolVersion:
		// we only support 3.0 as of now
		return errors.New("unsupported protocol version")
	}

	return nil
}

var _ backend.Server = (*Server)(nil)
