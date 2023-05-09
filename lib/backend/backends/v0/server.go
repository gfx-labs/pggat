package backends

import (
	"fmt"
	"log"
	"net"
	"runtime/debug"

	"pggat2/lib/auth/md5"
	"pggat2/lib/auth/sasl"
	"pggat2/lib/backend"
	"pggat2/lib/perror"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	"pggat2/lib/pnet/packet/packets/v3.0"
	"pggat2/lib/util/decorator"
)

var ErrServerFailed = perror.New(
	perror.ERROR,
	perror.InternalError,
	"Internal server error",
)

type Server struct {
	noCopy decorator.NoCopy

	conn net.Conn

	pnet.IOReader
	pnet.IOWriter

	cancellationKey [8]byte
	parameters      map[string]string
}

func NewServer(conn net.Conn) *Server {
	server := &Server{
		conn:       conn,
		IOReader:   pnet.MakeIOReader(conn),
		IOWriter:   pnet.MakeIOWriter(conn),
		parameters: make(map[string]string),
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
	key, value, ok := packets.ReadParameterStatus(in)
	if !ok {
		return pnet.ErrBadPacketFormat
	}
	T.parameters[key] = value
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

func (T *Server) fail() {
	log.Println("!!!!!!!!!!!!!!!!!!!!SERVER FAILED!!!!!!!!!!!!!!!!!!!!!!!!!!")
	debug.PrintStack()
}

func (T *Server) copyIn0(peer pnet.ReadWriter) (bool, perror.Error) {
	in, err := peer.Read()
	if err != nil {
		return false, perror.Wrap(err)
	}

	switch in.Type() {
	case packet.CopyData:
		err = pnet.ProxyPacket(T, in)
		if err != nil {
			T.fail()
			return false, ErrServerFailed
		}
		return false, nil
	case packet.CopyDone, packet.CopyFail:
		err = pnet.ProxyPacket(T, in)
		if err != nil {
			T.fail()
			return false, ErrServerFailed
		}
		return true, nil
	default:
		return false, pnet.ErrProtocolError
	}
}

func (T *Server) copyIn(peer pnet.ReadWriter, in packet.In) perror.Error {
	// send in (copyInResponse) to client
	err := pnet.ProxyPacket(peer, in)
	if err != nil {
		return perror.Wrap(err)
	}

	// copy in from client
	for {
		done, err := T.copyIn0(peer)
		if err != nil {
			// TODO(garet) return the server to a normal state
			return err
		}
		if done {
			break
		}
	}
	return nil
}

func (T *Server) copyOut0(peer pnet.ReadWriter) (bool, perror.Error) {
	in, err := T.Read()
	if err != nil {
		T.fail()
		return false, ErrServerFailed
	}

	switch in.Type() {
	case packet.CopyData:
		err = pnet.ProxyPacket(peer, in)
		return false, perror.Wrap(err)
	case packet.CopyDone, packet.ErrorResponse:
		err = pnet.ProxyPacket(peer, in)
		return true, perror.Wrap(err)
	default:
		T.fail()
		return false, ErrServerFailed
	}
}

func (T *Server) copyOut(peer pnet.ReadWriter, in packet.In) perror.Error {
	// send in (copyOutResponse) to server
	err := pnet.ProxyPacket(T, in)
	if err != nil {
		T.fail()
		return ErrServerFailed
	}

	// copy out from server
	for {
		done, err := T.copyOut0(peer)
		if err != nil {
			// TODO(garet) return the server to a normal state
			return err
		}
		if done {
			break
		}
	}
	return nil
}

func (T *Server) query0(peer pnet.ReadWriter) (bool, perror.Error) {
	in, err := T.Read()
	if err != nil {
		T.fail()
		return false, ErrServerFailed
	}
	switch in.Type() {
	case packet.CommandComplete,
		packet.RowDescription,
		packet.DataRow,
		packet.EmptyQueryResponse,
		packet.ErrorResponse,
		packet.NoticeResponse:
		err = pnet.ProxyPacket(peer, in)
		return false, perror.Wrap(err)
	case packet.CopyInResponse:
		err := T.copyIn(peer, in)
		return false, err
	case packet.CopyOutResponse:
		err := T.copyOut(peer, in)
		return false, err
	case packet.ReadyForQuery:
		err = pnet.ProxyPacket(peer, in)
		return true, perror.Wrap(err)
	case packet.ParameterStatus:
		err := T.parameterStatus(in)
		if err != nil {
			return false, err
		}
		return false, perror.Wrap(pnet.ProxyPacket(peer, in))
	default:
		T.fail()
		return false, ErrServerFailed
	}
}

func (T *Server) query(peer pnet.ReadWriter, in packet.In) perror.Error {
	// send in (initial query) to server
	err := pnet.ProxyPacket(T, in)
	if err != nil {
		T.fail()
		return ErrServerFailed
	}

	for {
		done, err := T.query0(peer)
		if err != nil {
			// TODO(garet) return to normal state
			return err
		}
		if done {
			break
		}
	}
	return nil
}

func (T *Server) functionCall0(peer pnet.ReadWriter) (bool, perror.Error) {
	in, err := T.Read()
	if err != nil {
		T.fail()
		return false, ErrServerFailed
	}

	switch in.Type() {
	case packet.ErrorResponse, packet.FunctionCallResponse, packet.NoticeResponse:
		err = pnet.ProxyPacket(peer, in)
		return false, perror.Wrap(err)
	case packet.ReadyForQuery:
		err = pnet.ProxyPacket(peer, in)
		return true, perror.Wrap(err)
	default:
		T.fail()
		return false, ErrServerFailed
	}
}

func (T *Server) functionCall(peer pnet.ReadWriter, in packet.In) perror.Error {
	// send in (FunctionCall) to server
	err := pnet.ProxyPacket(T, in)
	if err != nil {
		T.fail()
		return ErrServerFailed
	}

	for {
		done, err := T.functionCall0(peer)
		if err != nil {
			// TODO(garet) return to normal state
			return err
		}
		if done {
			break
		}
	}
	return nil
}

func (T *Server) handle(peer pnet.ReadWriter) perror.Error {
	in, err := peer.Read()
	if err != nil {
		return perror.Wrap(err)
	}
	switch in.Type() {
	case packet.Query:
		return T.query(peer, in)
	case packet.FunctionCall:
		return T.functionCall(peer, in)
	default:
		return perror.New(
			perror.FATAL,
			perror.FeatureNotSupported,
			"unsupported operation",
		)
	}
}

// Handle handles a transaction from peer, returning when the transaction is complete
func (T *Server) Handle(peer pnet.ReadWriter) {
	err := T.handle(peer)
	if err != nil {
		out := T.Write()
		packets.WriteErrorResponse(out, err)
		_ = out.Send()
	}
}

var _ backend.Server = (*Server)(nil)
