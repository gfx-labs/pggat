package frontends

import (
	"crypto/rand"
	"strings"

	"pggat2/lib/auth/sasl"
	"pggat2/lib/perror"
	"pggat2/lib/zap"
	"pggat2/lib/zap/packets/v3.0"
)

func startup0(client zap.ReadWriter, startupParameters map[string]string) (user, database string, done bool, err perror.Error) {
	packet := zap.NewUntypedPacket()
	defer packet.Done()
	err = perror.Wrap(client.ReadUntyped(packet))
	if err != nil {
		return
	}
	read := packet.Read()

	majorVersion, ok := read.ReadUint16()
	if !ok {
		err = packets.ErrBadFormat
		return
	}
	minorVersion, ok := read.ReadUint16()
	if !ok {
		err = packets.ErrBadFormat
		return
	}

	if majorVersion == 1234 {
		// Cancel or SSL
		switch minorVersion {
		case 5678:
			// Cancel
			err = perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"Cancel is not supported yet",
			)
			return
		case 5679:
			// SSL is not supported yet
			err = perror.Wrap(client.WriteByte('N'))
			return
		case 5680:
			// GSSAPI is not supported yet
			err = perror.Wrap(client.WriteByte('N'))
			return
		default:
			err = perror.New(
				perror.FATAL,
				perror.ProtocolViolation,
				"Unknown request code",
			)
			return
		}
	}

	if majorVersion != 3 {
		err = perror.New(
			perror.FATAL,
			perror.ProtocolViolation,
			"Unsupported protocol version",
		)
		return
	}

	var unsupportedOptions []string

	for {
		key, ok := read.ReadString()
		if !ok {
			err = packets.ErrBadFormat
			return
		}
		if key == "" {
			break
		}

		value, ok := read.ReadString()
		if !ok {
			err = packets.ErrBadFormat
			return
		}

		switch key {
		case "user":
			user = value
		case "database":
			database = value
		case "options":
			fields := strings.Fields(value)
			for i := 0; i < len(fields); i++ {
				switch fields[i] {
				case "-c":
					i++
					set := fields[i]
					key, value, ok = strings.Cut(set, "=")
					if !ok {
						err = perror.New(
							perror.FATAL,
							perror.ProtocolViolation,
							"Expected key=value",
						)
						return
					}

					startupParameters[key] = value
				default:
					err = perror.New(
						perror.FATAL,
						perror.FeatureNotSupported,
						"Flag not supported, sorry",
					)
					return
				}
			}
		case "replication":
			err = perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"Replication mode is not supported yet",
			)
			return
		default:
			if strings.HasPrefix(key, "_pq_.") {
				// we don't support protocol extensions at the moment
				unsupportedOptions = append(unsupportedOptions, key)
			} else {
				startupParameters[key] = value
			}
		}
	}

	if minorVersion != 0 || len(unsupportedOptions) > 0 {
		// negotiate protocol
		packet := zap.NewPacket()
		defer packet.Done()
		packets.WriteNegotiateProtocolVersion(packet, 0, unsupportedOptions)

		err = perror.Wrap(client.Write(packet))
		if err != nil {
			return
		}
	}

	if user == "" {
		err = perror.New(
			perror.FATAL,
			perror.InvalidAuthorizationSpecification,
			"User is required",
		)
		return
	}
	if database == "" {
		database = user
	}

	done = true
	return
}

func authenticationSASLInitial(client zap.ReadWriter, username, password string) (tool sasl.Server, resp []byte, done bool, err perror.Error) {
	// check which authentication method the client wants
	packet := zap.NewPacket()
	defer packet.Done()
	err = perror.Wrap(client.Read(packet))
	if err != nil {
		return
	}
	mechanism, initialResponse, ok := packets.ReadSASLInitialResponse(packet.Read())
	if !ok {
		err = packets.ErrBadFormat
		return
	}

	var err2 error
	tool, err2 = sasl.NewServer(mechanism, username, password)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}

	resp, done, err2 = tool.InitialResponse(initialResponse)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}
	return
}

func authenticationSASLContinue(client zap.ReadWriter, tool sasl.Server) (resp []byte, done bool, err perror.Error) {
	packet := zap.NewPacket()
	defer packet.Done()
	err = perror.Wrap(client.Read(packet))
	if err != nil {
		return
	}
	clientResp, ok := packets.ReadAuthenticationResponse(packet.Read())
	if !ok {
		err = packets.ErrBadFormat
		return
	}

	var err2 error
	resp, done, err2 = tool.Continue(clientResp)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}
	return
}

func authenticationSASL(client zap.ReadWriter, username, password string) perror.Error {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteAuthenticationSASL(packet, sasl.Mechanisms)
	err := perror.Wrap(client.Write(packet))
	if err != nil {
		return err
	}

	tool, resp, done, err := authenticationSASLInitial(client, username, password)
	if err != nil {
		return err
	}

	for {
		if done {
			packets.WriteAuthenticationSASLFinal(packet, resp)
			err = perror.Wrap(client.Write(packet))
			if err != nil {
				return err
			}
			break
		} else {
			packets.WriteAuthenticationSASLContinue(packet, resp)
			err = perror.Wrap(client.Write(packet))
			if err != nil {
				return err
			}
		}

		resp, done, err = authenticationSASLContinue(client, tool)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateParameter(pkts *zap.Packets, name, value string) {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteParameterStatus(packet, name, value)
	pkts.Append(packet)
}

func accept(client zap.ReadWriter, getPassword func(user, database string) (string, bool)) (user string, database string, startupParameters map[string]string, err perror.Error) {
	startupParameters = make(map[string]string)

	for {
		var done bool
		user, database, done, err = startup0(client, startupParameters)
		if err != nil {
			return
		}
		if done {
			break
		}
	}

	password, ok := getPassword(user, database)
	if !ok {
		err = perror.New(
			perror.FATAL,
			perror.InvalidPassword,
			"User or database not found",
		)
		return
	}

	err = authenticationSASL(client, user, password)
	if err != nil {
		return
	}

	pkts := zap.NewPackets()
	defer pkts.Done()

	// send auth Ok
	packet := zap.NewPacket()
	packets.WriteAuthenticationOk(packet)
	pkts.Append(packet)

	// send backend key data
	var cancellationKey [8]byte
	_, err2 := rand.Read(cancellationKey[:])
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}

	packet = zap.NewPacket()
	packets.WriteBackendKeyData(packet, cancellationKey)
	pkts.Append(packet)

	updateParameter(pkts, "client_encoding", "UTF8")
	updateParameter(pkts, "server_encoding", "UTF8")
	updateParameter(pkts, "server_version", "14.5")

	// send ready for query
	packet = zap.NewPacket()
	packets.WriteReadyForQuery(packet, 'I')
	pkts.Append(packet)

	err = perror.Wrap(client.WriteV(pkts))
	if err != nil {
		return
	}

	return
}

func fail(client zap.ReadWriter, err perror.Error) {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteErrorResponse(packet, err)
	_ = client.Write(packet)
}

func Accept(client zap.ReadWriter, getPassword func(user, database string) (string, bool)) (user, database string, startupParameters map[string]string, err perror.Error) {
	user, database, startupParameters, err = accept(client, getPassword)
	if err != nil {
		fail(client, err)
	}
	return
}
