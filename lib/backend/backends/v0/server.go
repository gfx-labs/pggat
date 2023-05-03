package backends

import (
	"errors"
	"log"
	"net"

	"pggat2/lib/auth/md5"
	"pggat2/lib/auth/sasl"
	"pggat2/lib/backend"
	"pggat2/lib/perror"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	"pggat2/lib/request"
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

func (T *Server) authenticationSASLChallenge(mechanism sasl.Client) (bool, error) {
	challenge, err := T.Read()
	if err != nil {
		return false, err
	}

	reader := packet.MakeReader(challenge)
	if reader.Type() != packet.Authentication {
		return false, ErrProtocolError
	}

	method, ok := reader.Int32()
	if !ok {
		return false, ErrBadPacketFormat
	}

	switch method {
	case 11:
		// challenge
		response, err := mechanism.Continue(reader.Remaining())
		if err != nil {
			return false, err
		}

		builder := packet.Builder{}
		builder.Type(packet.AuthenticationResponse)
		builder.Bytes(response)

		err = T.Write(builder.Raw())
		return false, err
	case 12:
		// finish
		err = mechanism.Final(reader.Remaining())
		if err != nil {
			return false, err
		}

		return true, nil
	default:
		return false, ErrProtocolError
	}
}

func (T *Server) authenticationSASL(mechanisms []string, username, password string) error {
	mechanism, err := sasl.NewClient(mechanisms, username, password)
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
	for {
		done, err := T.authenticationSASLChallenge(mechanism)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	return nil
}

func (T *Server) authenticationMD5(salt [4]byte, username, password string) error {
	var builder packet.Builder
	builder.Type(packet.AuthenticationResponse)
	builder.String(md5.Encode(username, password, salt))
	return T.Write(builder.Raw())
}

func (T *Server) authenticationCleartext(password string) error {
	var builder packet.Builder
	builder.Type(packet.AuthenticationResponse)
	builder.String(password)
	return T.Write(builder.Raw())
}

func (T *Server) startup0(username, password string) (bool, error) {
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
			return false, T.authenticationCleartext(password)
		case 5:
			salt, ok := reader.Bytes(4)
			if !ok {
				return false, ErrBadPacketFormat
			}
			return false, T.authenticationMD5([4]byte(salt), username, password)
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

			return false, T.authenticationSASL(mechanisms, username, password)
		default:
			return false, errors.New("unknown authentication method")
		}
	case packet.NegotiateProtocolVersion:
		// we only support protocol 3.0 for now
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
		T.cancellationKey = [8]byte(cancellationKey)
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
		// TODO(garet) don't hardcode username and password
		done, err := T.startup0("test", "password")
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

func (T *Server) simple() (bool, perror.Error) {
	pkt, err := T.Read()
	if err != nil {
		return false, perror.WrapError(err)
	}

	log.Printf("%#v", pkt)

	reader := packet.MakeReader(pkt)
	switch reader.Type() {
	case packet.CommandComplete,
		packet.RowDescription,
		packet.DataRow,
		packet.EmptyQueryResponse,
		packet.ErrorResponse,
		packet.NoticeResponse:
		return false, nil
	case packet.CopyInResponse:
		return false, nil
	case packet.CopyOutResponse:
		return false, nil
	case packet.ReadyForQuery:
		v, ok := reader.Uint8()
		if !ok {
			return false, perror.New(
				perror.FATAL,
				perror.ProtocolViolation,
				"Bad packet format",
			)
		}
		return v == 'I', nil
	default:
		return false, perror.New(
			perror.FATAL,
			perror.ProtocolViolation,
			"Unexpected packet",
		)
	}
}

func (T *Server) simpleRequest(req *request.Simple) perror.Error {
	// send forward
	err := T.Write(req.Query())
	if err != nil {
		return perror.WrapError(err)
	}

	for {
		done, perr := T.simple()
		if perr != nil {
			return perr
		}
		if done {
			break
		}
	}
	return nil
}

func (T *Server) Request(req request.Request) {
	switch v := req.(type) {
	case *request.Simple:
		T.simpleRequest(v)
	}
}

var _ backend.Server = (*Server)(nil)
