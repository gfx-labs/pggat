package backends

import (
	"fmt"
	"net"

	"pggat2/lib/auth/md5"
	"pggat2/lib/auth/sasl"
	"pggat2/lib/backend"
	"pggat2/lib/perror"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	"pggat2/lib/pnet/packet/packets/v3.0"
)

type Server struct {
	conn net.Conn

	reader pnet.IOReader
	writer pnet.IOWriter

	cancellationKey [8]byte
}

func NewServer(conn net.Conn) *Server {
	server := &Server{
		conn:   conn,
		reader: pnet.MakeIOReader(conn),
		writer: pnet.MakeIOWriter(conn),
	}
	err := server.accept()
	if err != nil {
		panic(fmt.Sprint("failed to connect to server: ", err))
		return nil
	}
	return server
}

func (T *Server) authenticationSASLChallenge(mechanism sasl.Client) (bool, perror.Error) {
	in, err := T.Read()
	if err != nil {
		return false, perror.Wrap(err)
	}

	if in.Type() != packet.Authentication {
		return false, pnet.ErrProtocolError
	}

	method, ok := in.Int32()
	if !ok {
		return false, pnet.ErrBadPacketFormat
	}

	switch method {
	case 11:
		// challenge
		response, err := mechanism.Continue(in.Remaining())
		if err != nil {
			return false, perror.Wrap(err)
		}

		out := T.Write()
		packets.WriteAuthenticationResponse(out, response)

		err = out.Send()
		return false, perror.Wrap(err)
	case 12:
		// finish
		err = mechanism.Final(in.Remaining())
		if err != nil {
			return false, perror.Wrap(err)
		}

		return true, nil
	default:
		return false, pnet.ErrProtocolError
	}
}

func (T *Server) authenticationSASL(mechanisms []string, username, password string) perror.Error {
	mechanism, err := sasl.NewClient(mechanisms, username, password)
	if err != nil {
		return perror.Wrap(err)
	}
	initialResponse := mechanism.InitialResponse()

	out := T.Write()
	packets.WriteSASLInitialResponse(out, mechanism.Name(), initialResponse)
	err = out.Send()
	if err != nil {
		return perror.Wrap(err)
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

func (T *Server) authenticationMD5(salt [4]byte, username, password string) perror.Error {
	out := T.Write()
	packets.WritePasswordMessage(out, md5.Encode(username, password, salt))
	return perror.Wrap(out.Send())
}

func (T *Server) authenticationCleartext(password string) perror.Error {
	out := T.Write()
	packets.WritePasswordMessage(out, password)
	return perror.Wrap(out.Send())
}

func (T *Server) startup0(username, password string) (bool, perror.Error) {
	in, err := T.Read()
	if err != nil {
		return false, perror.Wrap(err)
	}

	switch in.Type() {
	case packet.ErrorResponse:
		perr, ok := packets.ReadErrorResponse(in)
		if !ok {
			return false, pnet.ErrBadPacketFormat
		}
		return false, perr
	case packet.Authentication:
		method, ok := in.Int32()
		if !ok {
			return false, pnet.ErrBadPacketFormat
		}
		// they have more authentication methods than there are pokemon
		switch method {
		case 0:
			// we're good to go, that was easy
			return true, nil
		case 2:
			return false, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"kerberos v5 is not supported",
			)
		case 3:
			return false, T.authenticationCleartext(password)
		case 5:
			salt, ok := packets.ReadAuthenticationMD5(in)
			if !ok {
				return false, pnet.ErrBadPacketFormat
			}
			return false, T.authenticationMD5(salt, username, password)
		case 6:
			return false, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"scm credential is not supported",
			)
		case 7:
			return false, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"gss is not supported",
			)
		case 9:
			return false, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"sspi is not supported",
			)
		case 10:
			// read list of mechanisms
			mechanisms, ok := packets.ReadAuthenticationSASL(in)
			if !ok {
				return false, pnet.ErrBadPacketFormat
			}

			return false, T.authenticationSASL(mechanisms, username, password)
		default:
			return false, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"unknown authentication method",
			)
		}
	case packet.NegotiateProtocolVersion:
		// we only support protocol 3.0 for now
		return false, perror.New(
			perror.FATAL,
			perror.FeatureNotSupported,
			"server wanted to negotiate protocol version",
		)
	default:
		return false, pnet.ErrProtocolError
	}
}

func (T *Server) parameterStatus(in packet.In) perror.Error {
	// TODO(garet) do something with parameters
	return nil
}

func (T *Server) startup1() (bool, perror.Error) {
	in, err := T.Read()
	if err != nil {
		return false, perror.Wrap(err)
	}

	switch in.Type() {
	case packet.BackendKeyData:
		ok := in.Bytes(T.cancellationKey[:])
		if !ok {
			return false, pnet.ErrBadPacketFormat
		}
		return false, nil
	case packet.ParameterStatus:
		err := T.parameterStatus(in)
		return false, err
	case packet.ReadyForQuery:
		return true, nil
	case packet.ErrorResponse:
		err, ok := packets.ReadErrorResponse(in)
		if !ok {
			return false, pnet.ErrBadPacketFormat
		}
		return false, err
	case packet.NoticeResponse:
		// TODO(garet) do something with notice
		return false, nil
	default:
		return false, pnet.ErrProtocolError
	}
}

func (T *Server) accept() perror.Error {
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
		return perror.Wrap(err)
	}

	for {
		// TODO(garet) don't hardcode username and password
		done, err := T.startup0("postgres", "password")
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

func (T *Server) Write() packet.Out {
	return T.writer.Write()
}

func (T *Server) WriteByte(b byte) error {
	return T.writer.WriteByte(b)
}

func (T *Server) Send(typ packet.Type, payload []byte) error {
	return T.writer.Send(typ, payload)
}

func (T *Server) Read() (packet.In, error) {
	return T.reader.Read()
}

func (T *Server) ReadUntyped() (packet.In, error) {
	return T.reader.ReadUntyped()
}

var _ backend.Server = (*Server)(nil)
