package backends

import (
	"errors"
	"net"

	"pggat2/lib/auth/sasl"
	"pggat2/lib/backend"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
)

var ErrBadPacketFormat = errors.New("bad packet format")
var ErrProtocolError = errors.New("server sent unexpected packet")

type Server struct {
	conn net.Conn

	pnet.Reader
	pnet.Writer

	cancellationKey [8]byte
	parameters      map[string]string
}

func NewServer(conn net.Conn) (*Server, error) {
	server := &Server{
		conn:       conn,
		Reader:     pnet.MakeReader(conn),
		Writer:     pnet.MakeWriter(conn),
		parameters: make(map[string]string),
	}
	err := server.accept()
	if err != nil {
		return nil, err
	}
	return server, nil
}

func (T *Server) authenticationSASL(mechanisms []string) error {
	mechanism, err := sasl.NewClient(mechanisms, "test", "password")
	if err != nil {
		return err
	}

	builder := packet.Builder{}
	builder.Type(packet.AuthenticationResponse)
	builder.String(mechanism.Name())
	initialResponse := mechanism.InitialResponse()
	if initialResponse == nil {
		builder.Int32(-1)
	} else {
		builder.Int32(int32(len(initialResponse)))
		builder.Bytes(initialResponse)
	}
	err = T.Write(builder.Raw())
	if err != nil {
		return err
	}

	// challenge loop
outer:
	for {
		challenge, err := T.Read()
		if err != nil {
			return err
		}

		reader := packet.MakeReader(challenge)
		if reader.Type() != packet.Authentication {
			return ErrProtocolError
		}

		method, ok := reader.Int32()
		if !ok {
			return ErrBadPacketFormat
		}

		switch method {
		case 11:
			// challenge
			response, err := mechanism.Continue(reader.Remaining())
			if err != nil {
				return err
			}

			builder = packet.Builder{}
			builder.Type(packet.AuthenticationResponse)
			builder.Bytes(response)

			err = T.Write(builder.Raw())
			if err != nil {
				return err
			}
		case 12:
			// finish
			err = mechanism.Final(reader.Remaining())
			if err != nil {
				return err
			}

			break outer
		default:
			return ErrProtocolError
		}
	}

	return nil
}

func (T *Server) startup0() (bool, error) {
	pkt, err := T.Read()
	if err != nil {
		return false, err
	}

	reader := packet.MakeReader(pkt)
	switch reader.Type() {
	case packet.ErrorResponse:
		return false, errors.New("received error response")
	case packet.Authentication:
		method, ok := reader.Int32()
		if !ok {
			return false, ErrBadPacketFormat
		}
		// they have more authentication methods than there are pokemon
		switch method {
		case 0:
			// we're good to go, that was easy
			return true, nil
		case 2:
			return false, errors.New("kerberos v5 is not supported")
		case 3:
			return false, errors.New("cleartext is not supported")
		case 5:
			return false, errors.New("md5 password is not supported")
		case 6:
			return false, errors.New("scm credential is not supported")
		case 7:
			return false, errors.New("gss is not supported")
		case 9:
			return false, errors.New("sspi is not supported")
		case 10:
			// read list of mechanisms
			var mechanisms []string
			for {
				mechanism, ok := reader.String()
				if !ok {
					return false, ErrBadPacketFormat
				}
				if mechanism == "" {
					break
				}
				mechanisms = append(mechanisms, mechanism)
			}

			return false, T.authenticationSASL(mechanisms)
		default:
			// we only support protocol 3.0 for now
			return false, errors.New("unknown authentication method")
		}
	case packet.NegotiateProtocolVersion:
		return false, errors.New("server wanted to negotiate protocol version")
	default:
		return false, ErrProtocolError
	}
}

func (T *Server) startup1() (bool, error) {
	pkt, err := T.Read()
	if err != nil {
		return false, err
	}

	reader := packet.MakeReader(pkt)
	switch reader.Type() {
	case packet.BackendKeyData:
		cancellationKey, ok := reader.Bytes(8)
		if !ok {
			return false, ErrBadPacketFormat
		}
		copy(T.cancellationKey[:], cancellationKey)
		return false, nil
	case packet.ParameterStatus:
		parameter, ok := reader.String()
		if !ok {
			return false, ErrBadPacketFormat
		}
		value, ok := reader.String()
		if !ok {
			return false, ErrBadPacketFormat
		}
		T.parameters[parameter] = value
		return false, nil
	case packet.ReadyForQuery:
		return true, nil
	case packet.ErrorResponse:
		return false, errors.New("received error response")
	case packet.NoticeResponse:
		// TODO(garet) do something with notice
		return false, nil
	default:
		return false, ErrProtocolError
	}
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

	for {
		done, err := T.startup0()
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	for {
		done, err := T.startup1()
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	// startup complete, connection is ready for queries

	return nil
}

var _ backend.Server = (*Server)(nil)
