package backends

import (
	"errors"

	"pggat2/lib/auth"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func authenticationSASLChallenge(server zap.ReadWriter, encoder auth.SASLEncoder) (done bool, err error) {
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
		response, err = encoder.Write(read.ReadUnsafeRemaining())
		if err != nil {
			return
		}

		packets.WriteAuthenticationResponse(packet, response)

		err = server.Write(packet)
		return
	case 12:
		// finish
		_, err = encoder.Write(read.ReadUnsafeRemaining())
		if err != nil {
			return
		}

		return true, nil
	default:
		err = ErrUnexpectedPacket
		return
	}
}

func authenticationSASL(server zap.ReadWriter, mechanisms []string, creds auth.SASL) error {
	mechanism, encoder, err := creds.EncodeSASL(mechanisms)
	if err != nil {
		return err
	}
	initialResponse, err := encoder.Write(nil)
	if err != nil {
		return err
	}

	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteSASLInitialResponse(packet, mechanism, initialResponse)
	err = server.Write(packet)
	if err != nil {
		return err
	}

	// challenge loop
	for {
		var done bool
		done, err = authenticationSASLChallenge(server, encoder)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	return nil
}

func authenticationMD5(server zap.ReadWriter, salt [4]byte, creds auth.MD5) error {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WritePasswordMessage(packet, creds.EncodeMD5(salt))
	err := server.Write(packet)
	if err != nil {
		return err
	}
	return nil
}

func authenticationCleartext(server zap.ReadWriter, creds auth.Cleartext) error {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WritePasswordMessage(packet, creds.EncodeCleartext())
	err := server.Write(packet)
	if err != nil {
		return err
	}
	return nil
}

func startup0(server zap.ReadWriter, creds auth.Credentials) (done bool, err error) {
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
			c, ok := creds.(auth.Cleartext)
			if !ok {
				return false, auth.ErrMethodNotSupported
			}
			return false, authenticationCleartext(server, c)
		case 5:
			salt, ok := packets.ReadAuthenticationMD5(packet.Read())
			if !ok {
				err = ErrBadFormat
				return
			}
			c, ok := creds.(auth.MD5)
			if !ok {
				return false, auth.ErrMethodNotSupported
			}
			return false, authenticationMD5(server, salt, c)
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

			c, ok := creds.(auth.SASL)
			if !ok {
				return false, auth.ErrMethodNotSupported
			}
			return false, authenticationSASL(server, mechanisms, c)
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
		return
	}
}

func Accept(server zap.ReadWriter, creds auth.Credentials, database string, startupParameters map[string]string) error {
	if database == "" {
		database = creds.GetUsername()
	}
	// we can re-use the memory for this pkt most of the way down because we don't pass this anywhere
	packet := zap.NewUntypedPacket()
	defer packet.Done()
	packet.WriteInt16(3)
	packet.WriteInt16(0)
	packet.WriteString("user")
	packet.WriteString(creds.GetUsername())
	packet.WriteString("database")
	packet.WriteString(database)
	for key, value := range startupParameters {
		packet.WriteString(key)
		packet.WriteString(value)
	}
	packet.WriteString("")

	err := server.WriteUntyped(packet)
	if err != nil {
		return err
	}

	for {
		var done bool
		done, err = startup0(server, creds)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	for {
		var done bool
		done, err = startup1(server, startupParameters)
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
