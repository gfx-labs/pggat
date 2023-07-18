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
	packet := zap.NewPacket()
	defer packet.Done()
	err := server.Read(packet)
	if err != nil {
		fail(server, err)
		return false, Fail
	}
	read := packet.Read()

	if read.ReadType() != packets.Authentication {
		fail(server, ErrProtocolError)
		return false, Fail
	}

	method, ok := read.ReadInt32()
	if !ok {
		fail(server, ErrBadPacket)
		return false, Fail
	}

	switch method {
	case 11:
		// challenge
		response, err := mechanism.Continue(read.ReadUnsafeRemaining())
		if err != nil {
			fail(server, err)
			return false, Fail
		}

		packets.WriteAuthenticationResponse(packet, response)

		err = server.Write(packet)
		if err != nil {
			fail(server, err)
			return false, Fail
		}
		return false, Ok
	case 12:
		// finish
		err = mechanism.Final(read.ReadUnsafeRemaining())
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

	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteSASLInitialResponse(packet, mechanism.Name(), initialResponse)
	err = server.Write(packet)
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
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WritePasswordMessage(packet, md5.Encode(username, password, salt))
	err := server.Write(packet)
	if err != nil {
		fail(server, err)
		return Fail
	}
	return Ok
}

func authenticationCleartext(server zap.ReadWriter, password string) Status {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WritePasswordMessage(packet, password)
	err := server.Write(packet)
	if err != nil {
		fail(server, err)
		return Fail
	}
	return Ok
}

func startup0(server zap.ReadWriter, username, password string) (done bool, status Status) {
	packet := zap.NewPacket()
	defer packet.Done()
	err := server.Read(packet)
	if err != nil {
		fail(server, err)
		return false, Fail
	}
	read := packet.Read()

	switch read.ReadType() {
	case packets.ErrorResponse:
		perr, ok := packets.ReadErrorResponse(&read)
		if !ok {
			fail(server, ErrBadPacket)
			return false, Fail
		}
		failpg(server, perr)
		return false, Fail
	case packets.Authentication:
		read2 := read
		method, ok := read2.ReadInt32()
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
			salt, ok := packets.ReadAuthenticationMD5(&read)
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
			mechanisms, ok := packets.ReadAuthenticationSASL(&read)
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
	packet := zap.NewPacket()
	defer packet.Done()
	err := server.Read(packet)
	if err != nil {
		fail(server, err)
		return false, Fail
	}
	read := packet.Read()

	switch read.ReadType() {
	case packets.BackendKeyData:
		var cancellationKey [8]byte
		ok := read.ReadBytes(cancellationKey[:])
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
		perr, ok := packets.ReadErrorResponse(&read)
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
	packet := zap.NewUntypedPacket()
	packet.WriteInt16(3)
	packet.WriteInt16(0)
	packet.WriteString("user")
	packet.WriteString(username)
	packet.WriteString("database")
	packet.WriteString(database)
	packet.WriteString("")

	err := server.WriteUntyped(packet)
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
