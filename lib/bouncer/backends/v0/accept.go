package backends

import (
	"pggat2/lib/auth/md5"
	"pggat2/lib/auth/sasl"
	"pggat2/lib/perror"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func authenticationSASLChallenge(server zap.ReadWriter, mechanism sasl.Client) (done bool, err perror.Error) {
	packet := zap.NewPacket()
	defer packet.Done()
	err = perror.Wrap(server.Read(packet))
	if err != nil {
		return
	}
	read := packet.Read()

	if read.ReadType() != packets.Authentication {
		err = packets.ErrUnexpectedPacket
		return
	}

	method, ok := read.ReadInt32()
	if !ok {
		err = packets.ErrBadFormat
		return
	}

	switch method {
	case 11:
		// challenge
		response, err2 := mechanism.Continue(read.ReadUnsafeRemaining())
		if err2 != nil {
			err = perror.Wrap(err2)
			return
		}

		packets.WriteAuthenticationResponse(packet, response)

		err = perror.Wrap(server.Write(packet))
		return
	case 12:
		// finish
		err = perror.Wrap(mechanism.Final(read.ReadUnsafeRemaining()))
		if err != nil {
			return
		}

		return true, nil
	default:
		err = packets.ErrUnexpectedPacket
		return
	}
}

func authenticationSASL(server zap.ReadWriter, mechanisms []string, username, password string) perror.Error {
	mechanism, err := sasl.NewClient(mechanisms, username, password)
	if err != nil {
		return perror.Wrap(err)
	}
	initialResponse := mechanism.InitialResponse()

	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteSASLInitialResponse(packet, mechanism.Name(), initialResponse)
	err = server.Write(packet)
	if err != nil {
		return perror.Wrap(err)
	}

	// challenge loop
	for {
		done, err := authenticationSASLChallenge(server, mechanism)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	return nil
}

func authenticationMD5(server zap.ReadWriter, salt [4]byte, username, password string) perror.Error {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WritePasswordMessage(packet, md5.Encode(username, password, salt))
	err := server.Write(packet)
	if err != nil {
		return perror.Wrap(err)
	}
	return nil
}

func authenticationCleartext(server zap.ReadWriter, password string) perror.Error {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WritePasswordMessage(packet, password)
	err := server.Write(packet)
	if err != nil {
		return perror.Wrap(err)
	}
	return nil
}

func startup0(server zap.ReadWriter, username, password string) (done bool, err perror.Error) {
	packet := zap.NewPacket()
	defer packet.Done()
	err = perror.Wrap(server.Read(packet))
	if err != nil {
		return
	}
	read := packet.Read()

	switch read.ReadType() {
	case packets.ErrorResponse:
		var ok bool
		err, ok = packets.ReadErrorResponse(&read)
		if !ok {
			err = packets.ErrBadFormat
		}
		return
	case packets.Authentication:
		read2 := read
		method, ok := read2.ReadInt32()
		if !ok {
			err = packets.ErrBadFormat
			return
		}
		// they have more authentication methods than there are pokemon
		switch method {
		case 0:
			// we're good to go, that was easy
			return true, nil
		case 2:
			err = perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"kerberos v5 is not supported",
			)
			return
		case 3:
			return false, authenticationCleartext(server, password)
		case 5:
			salt, ok := packets.ReadAuthenticationMD5(&read)
			if !ok {
				err = packets.ErrBadFormat
				return
			}
			return false, authenticationMD5(server, salt, username, password)
		case 6:
			err = perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"scm credential is not supported",
			)
			return
		case 7:
			err = perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"gss is not supported",
			)
			return
		case 9:
			err = perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"sspi is not supported",
			)
			return
		case 10:
			// read list of mechanisms
			mechanisms, ok := packets.ReadAuthenticationSASL(&read)
			if !ok {
				err = packets.ErrBadFormat
				return
			}

			return false, authenticationSASL(server, mechanisms, username, password)
		default:
			err = perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"unknown authentication method",
			)
			return
		}
	case packets.NegotiateProtocolVersion:
		// we only support protocol 3.0 for now
		err = perror.New(
			perror.FATAL,
			perror.FeatureNotSupported,
			"server wanted to negotiate protocol version",
		)
		return
	default:
		err = packets.ErrUnexpectedPacket
		return
	}
}

func startup1(server zap.ReadWriter) (done bool, err perror.Error) {
	packet := zap.NewPacket()
	defer packet.Done()
	err = perror.Wrap(server.Read(packet))
	if err != nil {
		return
	}
	read := packet.Read()

	switch read.ReadType() {
	case packets.BackendKeyData:
		var cancellationKey [8]byte
		ok := read.ReadBytes(cancellationKey[:])
		if !ok {
			err = packets.ErrBadFormat
			return
		}
		// TODO(garet) put cancellation key somewhere
		return false, nil
	case packets.ParameterStatus:
		return false, nil
	case packets.ReadyForQuery:
		return true, nil
	case packets.ErrorResponse:
		var ok bool
		err, ok = packets.ReadErrorResponse(&read)
		if !ok {
			err = packets.ErrBadFormat
		}
		return
	case packets.NoticeResponse:
		// TODO(garet) do something with notice
		return false, nil
	default:
		err = packets.ErrUnexpectedPacket
		return false, err
	}
}

func Accept(server zap.ReadWriter, username, password, database string) perror.Error {
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

	err := perror.Wrap(server.WriteUntyped(packet))
	if err != nil {
		return err
	}

	for {
		done, err := startup0(server, username, password)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	for {
		done, err := startup1(server)
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
