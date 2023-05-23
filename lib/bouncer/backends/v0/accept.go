package backends

import (
	"errors"

	"pggat2/lib/auth/md5"
	"pggat2/lib/auth/sasl"
	"pggat2/lib/perror"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Status int

const (
	Fail Status = iota
	Ok
)

var (
	ErrProtocolError = errors.New("protocol error")
	ErrBadPacket     = errors.New("bad packet")
)

func fail(server zap.ReadWriter, err error) {
	panic(err)
}

func failpg(server zap.ReadWriter, err perror.Error) {
	panic(err)
}

func authenticationSASLChallenge(server zap.ReadWriter, mechanism sasl.Client) (done bool, status Status) {
	in, err := server.Read()
	if err != nil {
		fail(server, err)
		return false, Fail
	}

	if in.Type() != packets.Authentication {
		fail(server, ErrProtocolError)
		return false, Fail
	}

	method, ok := in.Int32()
	if !ok {
		fail(server, ErrBadPacket)
		return false, Fail
	}

	switch method {
	case 11:
		// challenge
		response, err := mechanism.Continue(in.Remaining())
		if err != nil {
			fail(server, err)
			return false, Fail
		}

		out := server.Write()
		packets.WriteAuthenticationResponse(out, response)

		err = server.Send(out)
		if err != nil {
			fail(server, err)
			return false, Fail
		}
		return false, Ok
	case 12:
		// finish
		err = mechanism.Final(in.Remaining())
		if err != nil {
			fail(server, err)
			return false, Fail
		}

		return true, Ok
	default:
		fail(server, ErrProtocolError)
		return false, Fail
	}
}

func authenticationSASL(server zap.ReadWriter, mechanisms []string, username, password string) Status {
	mechanism, err := sasl.NewClient(mechanisms, username, password)
	if err != nil {
		fail(server, err)
		return Fail
	}
	initialResponse := mechanism.InitialResponse()

	out := server.Write()
	packets.WriteSASLInitialResponse(out, mechanism.Name(), initialResponse)
	err = server.Send(out)
	if err != nil {
		fail(server, err)
		return Fail
	}

	// challenge loop
	for {
		done, status := authenticationSASLChallenge(server, mechanism)
		if status != Ok {
			return status
		}
		if done {
			break
		}
	}

	return Ok
}

func authenticationMD5(server zap.ReadWriter, salt [4]byte, username, password string) Status {
	out := server.Write()
	packets.WritePasswordMessage(out, md5.Encode(username, password, salt))
	err := server.Send(out)
	if err != nil {
		fail(server, err)
		return Fail
	}
	return Ok
}

func authenticationCleartext(server zap.ReadWriter, password string) Status {
	out := server.Write()
	packets.WritePasswordMessage(out, password)
	err := server.Send(out)
	if err != nil {
		fail(server, err)
		return Fail
	}
	return Ok
}

func startup0(server zap.ReadWriter, username, password string) (done bool, status Status) {
	in, err := server.Read()
	if err != nil {
		fail(server, err)
		return false, Fail
	}

	switch in.Type() {
	case packets.ErrorResponse:
		perr, ok := packets.ReadErrorResponse(in)
		if !ok {
			fail(server, ErrBadPacket)
			return false, Fail
		}
		failpg(server, perr)
		return false, Fail
	case packets.Authentication:
		method, ok := in.Int32()
		if !ok {
			fail(server, ErrBadPacket)
			return false, Fail
		}
		// they have more authentication methods than there are pokemon
		switch method {
		case 0:
			// we're good to go, that was easy
			return true, Ok
		case 2:
			fail(server, errors.New("kerberos v5 is not supported"))
			return false, Fail
		case 3:
			return false, authenticationCleartext(server, password)
		case 5:
			salt, ok := packets.ReadAuthenticationMD5(in)
			if !ok {
				fail(server, ErrBadPacket)
				return false, Fail
			}
			return false, authenticationMD5(server, salt, username, password)
		case 6:
			fail(server, errors.New("scm credential is not supported"))
			return false, Fail
		case 7:
			fail(server, errors.New("gss is not supported"))
			return false, Fail
		case 9:
			fail(server, errors.New("sspi is not supported"))
			return false, Fail
		case 10:
			// read list of mechanisms
			mechanisms, ok := packets.ReadAuthenticationSASL(in)
			if !ok {
				fail(server, ErrBadPacket)
				return false, Fail
			}

			return false, authenticationSASL(server, mechanisms, username, password)
		default:
			fail(server, errors.New("unknown authentication method"))
			return false, Fail
		}
	case packets.NegotiateProtocolVersion:
		// we only support protocol 3.0 for now
		fail(server, errors.New("server wanted to negotiate protocol version"))
		return false, Fail
	default:
		fail(server, ErrProtocolError)
		return false, Fail
	}
}

func startup1(server zap.ReadWriter) (done bool, status Status) {
	in, err := server.Read()
	if err != nil {
		fail(server, err)
		return false, Fail
	}

	switch in.Type() {
	case packets.BackendKeyData:
		var cancellationKey [8]byte
		ok := in.Bytes(cancellationKey[:])
		if !ok {
			fail(server, ErrBadPacket)
			return false, Fail
		}
		// TODO(garet) put cancellation key somewhere
		return false, Ok
	case packets.ParameterStatus:
		return false, Ok
	case packets.ReadyForQuery:
		return true, Ok
	case packets.ErrorResponse:
		perr, ok := packets.ReadErrorResponse(in)
		if !ok {
			fail(server, ErrBadPacket)
			return false, Fail
		}
		failpg(server, perr)
		return false, Fail
	case packets.NoticeResponse:
		// TODO(garet) do something with notice
		return false, Ok
	default:
		fail(server, ErrProtocolError)
		return false, Fail
	}
}

func Accept(server zap.ReadWriter, username, password, database string) {
	if database == "" {
		database = username
	}
	// we can re-use the memory for this pkt most of the way down because we don't pass this anywhere
	out := server.Write()
	out.Int16(3)
	out.Int16(0)
	out.String("user")
	out.String(username)
	out.String("database")
	out.String(database)
	out.String("")

	err := server.Send(out)
	if err != nil {
		fail(server, err)
		return
	}

	for {
		done, status := startup0(server, username, password)
		if status != Ok {
			return
		}
		if done {
			break
		}
	}

	for {
		done, status := startup1(server)
		if status != Ok {
			return
		}
		if done {
			break
		}
	}

	// startup complete, connection is ready for queries
}
