package backends

import (
	"errors"

	"pggat2/lib/auth"
	"pggat2/lib/bouncer"
	"pggat2/lib/util/strutil"
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

func startup1(conn *bouncer.Conn) (done bool, err error) {
	packet := zap.NewPacket()
	defer packet.Done()
	err = conn.RW.Read(packet)
	if err != nil {
		return
	}

	switch packet.ReadType() {
	case packets.BackendKeyData:
		read := packet.Read()
		ok := read.ReadBytes(conn.BackendKey[:])
		if !ok {
			err = ErrBadFormat
			return
		}
		return false, nil
	case packets.ParameterStatus:
		key, value, ok := packets.ReadParameterStatus(packet.Read())
		if !ok {
			err = ErrBadFormat
			return
		}
		ikey := strutil.MakeCIString(key)
		if conn.InitialParameters == nil {
			conn.InitialParameters = make(map[strutil.CIString]string)
		}
		conn.InitialParameters[ikey] = value
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

func Accept(server zap.ReadWriter, options AcceptOptions) (bouncer.Conn, error) {
	username := options.Credentials.GetUsername()

	if options.Database == "" {
		options.Database = username
	}

	// we can re-use the memory for this pkt most of the way down because we don't pass this anywhere
	packet := zap.NewUntypedPacket()
	defer packet.Done()
	packet.WriteInt16(3)
	packet.WriteInt16(0)
	packet.WriteString("user")
	packet.WriteString(username)
	packet.WriteString("database")
	packet.WriteString(options.Database)
	for key, value := range options.StartupParameters {
		packet.WriteString(key.String())
		packet.WriteString(value)
	}
	packet.WriteString("")

	err := server.WriteUntyped(packet)
	if err != nil {
		return bouncer.Conn{}, err
	}

	for {
		var done bool
		done, err = startup0(server, options.Credentials)
		if err != nil {
			return bouncer.Conn{}, err
		}
		if done {
			break
		}
	}

	conn := bouncer.Conn{
		RW:       server,
		User:     username,
		Database: options.Database,
	}

	for {
		var done bool
		done, err = startup1(&conn)
		if err != nil {
			return bouncer.Conn{}, err
		}
		if done {
			break
		}
	}

	// startup complete, connection is ready for queries
	return conn, nil
}
