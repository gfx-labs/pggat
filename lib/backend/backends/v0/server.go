package backends

import (
	"errors"
	"net"

	"pggat2/lib/auth/md5"
	"pggat2/lib/auth/sasl"
	"pggat2/lib/backend"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	"pggat2/lib/util/decorator"
)

var ErrBadPacketFormat = errors.New("bad packet format")
var ErrProtocolError = errors.New("server sent unexpected packet")

type Server struct {
	noCopy decorator.NoCopy

	conn net.Conn

	pnet.IOReader
	pnet.IOWriter

	cancellationKey [8]byte
	parameters      map[string]string
}

func NewServer(conn net.Conn) (*Server, error) {
	server := &Server{
		conn:       conn,
		IOReader:   pnet.MakeIOReader(conn),
		IOWriter:   pnet.MakeIOWriter(conn),
		parameters: make(map[string]string),
	}
	err := server.accept()
	if err != nil {
		return nil, err
	}
	return server, nil
}

func (T *Server) authenticationSASLChallenge(mechanism sasl.Client) (bool, error) {
	in, err := T.Read()
	if err != nil {
		return false, err
	}

	if in.Type() != packet.Authentication {
		return false, ErrProtocolError
	}

	method, ok := in.Int32()
	if !ok {
		return false, ErrBadPacketFormat
	}

	switch method {
	case 11:
		// challenge
		response, err := mechanism.Continue(in.Remaining())
		if err != nil {
			return false, err
		}

		out := T.Write()
		out.Type(packet.AuthenticationResponse)
		out.Bytes(response)

		err = out.Send()
		return false, err
	case 12:
		// finish
		err = mechanism.Final(in.Remaining())
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
	initialResponse := mechanism.InitialResponse()

	out := T.Write()
	out.Type(packet.AuthenticationResponse)
	out.String(mechanism.Name())
	if initialResponse == nil {
		out.Int32(-1)
	} else {
		out.Int32(int32(len(initialResponse)))
		out.Bytes(initialResponse)
	}
	err = out.Send()
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
	out := T.Write()
	out.Type(packet.AuthenticationResponse)
	out.String(md5.Encode(username, password, salt))
	return out.Send()
}

func (T *Server) authenticationCleartext(password string) error {
	out := T.Write()
	out.Type(packet.AuthenticationResponse)
	out.String(password)
	return out.Send()
}

func (T *Server) startup0(username, password string) (bool, error) {
	in, err := T.Read()
	if err != nil {
		return false, err
	}

	switch in.Type() {
	case packet.ErrorResponse:
		return false, errors.New("received error response")
	case packet.Authentication:
		method, ok := in.Int32()
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
			var salt [4]byte
			ok = in.Bytes(salt[:])
			if !ok {
				return false, ErrBadPacketFormat
			}
			return false, T.authenticationMD5(salt, username, password)
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
				mechanism, ok := in.String()
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
	in, err := T.Read()
	if err != nil {
		return false, err
	}

	switch in.Type() {
	case packet.BackendKeyData:
		ok := in.Bytes(T.cancellationKey[:])
		if !ok {
			return false, ErrBadPacketFormat
		}
		return false, nil
	case packet.ParameterStatus:
		parameter, ok := in.String()
		if !ok {
			return false, ErrBadPacketFormat
		}
		value, ok := in.String()
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
	// we can re-use the memory for this pkt most of the way down because we don't pass this anywhere
	out := T.Write()
	out.Int16(3)
	out.Int16(0)
	// TODO(garet) don't hardcode username and password
	out.String("user")
	out.String("postgres")
	out.String("")

	err := out.Send()
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

func (T *Server) proxy(in *packet.In) error {
	out := T.Write()
	out.Type(in.Type())
	out.Bytes(in.Full())
	return out.Send()
}

func (T *Server) query(peer pnet.ReadWriter) error {
	return nil
}

// Transaction handles a transaction from peer, returning when the transaction is complete
func (T *Server) Transaction(peer pnet.ReadWriter) error {
	in, err := peer.Read()
	if err != nil {
		return err
	}
	switch in.Type() {
	case packet.Query:
		// proxy to backend
		err = T.proxy(&in)
		if err != nil {
			return err
		}
		return T.query(peer)
	default:
		return errors.New("unsupported operation")
	}
}

var _ backend.Server = (*Server)(nil)
