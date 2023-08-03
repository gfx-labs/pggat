package backends

import (
	"errors"

	"pggat2/lib/auth/md5"
	"pggat2/lib/auth/sasl"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func authenticationSASLChallenge(server zap.ReadWriter, mechanism sasl.Client) (done bool, err error) {
	packet := zap.NewPacket()
	defer packet.Done()
	err = server.Read(packet)
	if err != nil {
		return
	}
	read := packet.Read()

	if read.ReadType() != packets.Authentication {
		err = ErrUnexpectedPacket
		return
	}

	method, ok := read.ReadInt32()
	if !ok {
		err = ErrBadFormat
		return
	}

	switch method {
	case 11:
		// challenge
		var response []byte
		response, err = mechanism.Continue(read.ReadUnsafeRemaining())
		if err != nil {
			return
		}

		packets.WriteAuthenticationResponse(packet, response)

		err = server.Write(packet)
		return
	case 12:
		// finish
		err = mechanism.Final(read.ReadUnsafeRemaining())
		if err != nil {
			return
		}

		return true, nil
	default:
		err = ErrUnexpectedPacket
		return
	}
}

func authenticationSASL(server zap.ReadWriter, mechanisms []string, username, password string) error {
	mechanism, err := sasl.NewClient(mechanisms, username, password)
	if err != nil {
		return err
	}
	initialResponse := mechanism.InitialResponse()

	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteSASLInitialResponse(packet, mechanism.Name(), initialResponse)
	err = server.Write(packet)
	if err != nil {
		return err
	}

	// challenge loop
	for {
		var done bool
		done, err = authenticationSASLChallenge(server, mechanism)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	return nil
}

func authenticationMD5(server zap.ReadWriter, salt [4]byte, username, password string) error {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WritePasswordMessage(packet, md5.Encode(username, password, salt))
	err := server.Write(packet)
	if err != nil {
		return err
	}
	return nil
}

func authenticationCleartext(server zap.ReadWriter, password string) error {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WritePasswordMessage(packet, password)
	err := server.Write(packet)
	if err != nil {
		return err
	}
	return nil
}

func startup0(server zap.ReadWriter, username, password string) (done bool, err error) {
	packet := zap.NewPacket()
	defer packet.Done()
	err = server.Read(packet)
	if err != nil {
		return
	}

	switch packet.ReadType() {
	case packets.ErrorResponse:
		err2, ok := packets.ReadErrorResponse(packet.Read())
		if !ok {
			err = ErrBadFormat
		} else {
			err = errors.New(err2.String())
		}
		return
	case packets.Authentication:
		read := packet.Read()
		method, ok := read.ReadInt32()
		if !ok {
			err = ErrBadFormat
			return
		}
		// they have more authentication methods than there are pokemon
		switch method {
		case 0:
			// we're good to go, that was easy
			return true, nil
		case 2:
			err = errors.New("kerberos v5 is not supported")
			return
		case 3:
			return false, authenticationCleartext(server, password)
		case 5:
			salt, ok := packets.ReadAuthenticationMD5(packet.Read())
			if !ok {
				err = ErrBadFormat
				return
			}
			return false, authenticationMD5(server, salt, username, password)
		case 6:
			err = errors.New("scm credential is not supported")
			return
		case 7:
			err = errors.New("gss is not supported")
			return
		case 9:
			err = errors.New("sspi is not supported")
			return
		case 10:
			// read list of mechanisms
			mechanisms, ok := packets.ReadAuthenticationSASL(packet.Read())
			if !ok {
				err = ErrBadFormat
				return
			}

			return false, authenticationSASL(server, mechanisms, username, password)
		default:
			err = errors.New("unknown authentication method")
			return
		}
	case packets.NegotiateProtocolVersion:
		// we only support protocol 3.0 for now
		err = errors.New("server wanted to negotiate protocol version")
		return
	default:
		err = ErrUnexpectedPacket
		return
	}
}

func startup1(server zap.ReadWriter, parameterStatus map[string]string) (done bool, err error) {
	packet := zap.NewPacket()
	defer packet.Done()
	err = server.Read(packet)
	if err != nil {
		return
	}

	switch packet.ReadType() {
	case packets.BackendKeyData:
		read := packet.Read()
		var cancellationKey [8]byte
		ok := read.ReadBytes(cancellationKey[:])
		if !ok {
			err = ErrBadFormat
			return
		}
		// TODO(garet) put cancellation key somewhere
		return false, nil
	case packets.ParameterStatus:
		key, value, ok := packets.ReadParameterStatus(packet.Read())
		if !ok {
			err = ErrBadFormat
			return
		}
		parameterStatus[key] = value
		return false, nil
	case packets.ReadyForQuery:
		return true, nil
	case packets.ErrorResponse:
		err2, ok := packets.ReadErrorResponse(packet.Read())
		if !ok {
			err = ErrBadFormat
		} else {
			err = errors.New(err2.String())
		}
		return
	case packets.NoticeResponse:
		// TODO(garet) do something with notice
		return false, nil
	default:
		err = ErrUnexpectedPacket
		return false, err
	}
}

func Accept(server zap.ReadWriter, username, password, database string) (map[string]string, error) {
	parameterStatus := make(map[string]string)

	if database == "" {
		database = username
	}
	// we can re-use the memory for this pkt most of the way down because we don't pass this anywhere
	packet := zap.NewUntypedPacket()
	defer packet.Done()
	packet.WriteInt16(3)
	packet.WriteInt16(0)
	packet.WriteString("user")
	packet.WriteString(username)
	packet.WriteString("database")
	packet.WriteString(database)
	packet.WriteString("")

	err := server.WriteUntyped(packet)
	if err != nil {
		return nil, err
	}

	for {
		var done bool
		done, err = startup0(server, username, password)
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
	}

	for {
		var done bool
		done, err = startup1(server, parameterStatus)
		if err != nil {
			return nil, err
		}
		if done {
			break
		}
	}

	// startup complete, connection is ready for queries
	return parameterStatus, nil
}
