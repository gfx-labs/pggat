package frontends

import (
	"crypto/rand"
	"strings"

	"pggat2/lib/auth/sasl"
	"pggat2/lib/perror"
	"pggat2/lib/zap"
	"pggat2/lib/zap/packets/v3.0"
)

type Status int

const (
	Fail Status = iota
	Ok
)

func fail(client zap.ReadWriter, err perror.Error) {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteErrorResponse(packet, err)
	_ = client.Write(packet)
}

func startup0(client zap.ReadWriter) (user, database string, done bool, status Status) {
	packet := zap.NewUntypedPacket()
	defer packet.Done()
	err := client.ReadUntyped(packet)
	if err != nil {
		fail(client, perror.Wrap(err))
		return
	}
	read := packet.Read()

	majorVersion, ok := read.ReadUint16()
	if !ok {
		fail(client, packets.ErrBadFormat)
		return
	}
	minorVersion, ok := read.ReadUint16()
	if !ok {
		fail(client, packets.ErrBadFormat)
		return
	}

	if majorVersion == 1234 {
		// Cancel or SSL
		switch minorVersion {
		case 5678:
			// Cancel
			fail(client, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"Cancel is not supported yet",
			))
			return
		case 5679:
			// SSL is not supported yet
			err = client.WriteByte('N')
			if err != nil {
				fail(client, perror.Wrap(err))
				return
			}
			status = Ok
			return
		case 5680:
			// GSSAPI is not supported yet
			err = client.WriteByte('N')
			if err != nil {
				fail(client, perror.Wrap(err))
				return
			}
			status = Ok
			return
		default:
			fail(client, perror.New(
				perror.FATAL,
				perror.ProtocolViolation,
				"Unknown request code",
			))
			return
		}
	}

	if majorVersion != 3 {
		fail(client, perror.New(
			perror.FATAL,
			perror.ProtocolViolation,
			"Unsupported protocol version",
		))
	}

	var unsupportedOptions []string

	for {
		key, ok := read.ReadString()
		if !ok {
			fail(client, packets.ErrBadFormat)
			return
		}
		if key == "" {
			break
		}

		value, ok := read.ReadString()
		if !ok {
			fail(client, packets.ErrBadFormat)
			return
		}

		switch key {
		case "user":
			user = value
		case "database":
			database = value
		case "options":
			fail(client, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"Startup options are not supported yet",
			))
			return
		case "replication":
			fail(client, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"Replication mode is not supported yet",
			))
			return
		default:
			if strings.HasPrefix(key, "_pq_.") {
				// we don't support protocol extensions at the moment
				unsupportedOptions = append(unsupportedOptions, key)
			} else {
				// TODO(garet) do something with this parameter
			}
		}
	}

	if minorVersion != 0 || len(unsupportedOptions) > 0 {
		// negotiate protocol
		packet := zap.NewPacket()
		defer packet.Done()
		packets.WriteNegotiateProtocolVersion(packet, 0, unsupportedOptions)

		err = client.Write(packet)
		if err != nil {
			fail(client, perror.Wrap(err))
			return
		}
	}

	if user == "" {
		fail(client, perror.New(
			perror.FATAL,
			perror.InvalidAuthorizationSpecification,
			"User is required",
		))
		return
	}
	if database == "" {
		database = user
	}

	status = Ok
	done = true
	return
}

func authenticationSASLInitial(client zap.ReadWriter, username, password string) (server sasl.Server, resp []byte, done bool, status Status) {
	// check which authentication method the client wants
	packet := zap.NewPacket()
	defer packet.Done()
	err := client.Read(packet)
	if err != nil {
		fail(client, perror.Wrap(err))
		return nil, nil, false, Fail
	}
	read := packet.Read()
	mechanism, initialResponse, ok := packets.ReadSASLInitialResponse(&read)
	if !ok {
		fail(client, packets.ErrBadFormat)
		return nil, nil, false, Fail
	}

	tool, err := sasl.NewServer(mechanism, username, password)
	if err != nil {
		fail(client, perror.Wrap(err))
		return nil, nil, false, Fail
	}

	resp, done, err = tool.InitialResponse(initialResponse)
	if err != nil {
		fail(client, perror.Wrap(err))
		return nil, nil, false, Fail
	}
	return tool, resp, done, Ok
}

func authenticationSASLContinue(client zap.ReadWriter, tool sasl.Server) (resp []byte, done bool, status Status) {
	packet := zap.NewPacket()
	defer packet.Done()
	err := client.Read(packet)
	if err != nil {
		fail(client, perror.Wrap(err))
		return nil, false, Fail
	}
	read := packet.Read()
	clientResp, ok := packets.ReadAuthenticationResponse(&read)
	if !ok {
		fail(client, packets.ErrBadFormat)
		return nil, false, Fail
	}

	resp, done, err = tool.Continue(clientResp)
	if err != nil {
		fail(client, perror.Wrap(err))
		return nil, false, Fail
	}
	return resp, done, Ok
}

func authenticationSASL(client zap.ReadWriter, username, password string) Status {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteAuthenticationSASL(packet, sasl.Mechanisms)
	err := client.Write(packet)
	if err != nil {
		fail(client, perror.Wrap(err))
		return Fail
	}

	tool, resp, done, status := authenticationSASLInitial(client, username, password)

	for {
		if status != Ok {
			return status
		}
		if done {
			packets.WriteAuthenticationSASLFinal(packet, resp)
			err = client.Write(packet)
			if err != nil {
				fail(client, perror.Wrap(err))
				return Fail
			}
			break
		} else {
			packets.WriteAuthenticationSASLContinue(packet, resp)
			err = client.Write(packet)
			if err != nil {
				fail(client, perror.Wrap(err))
				return Fail
			}
		}

		resp, done, status = authenticationSASLContinue(client, tool)
	}

	return Ok
}

func updateParameter(pkts *zap.Packets, name, value string) Status {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteParameterStatus(packet, name, value)
	pkts.Append(packet)
	return Ok
}

func Accept(client zap.ReadWriter, getPassword func(user string, database string) string, initialParameterStatus map[string]string) (user string, database string, ok bool) {
	for {
		var done bool
		var status Status
		user, database, done, status = startup0(client)
		if status != Ok {
			return
		}
		if done {
			break
		}
	}

	status := authenticationSASL(client, user, getPassword(user, database))
	if status != Ok {
		return
	}

	pkts := zap.NewPackets()
	defer pkts.Done()

	// send auth Ok
	packet := zap.NewPacket()
	packets.WriteAuthenticationOk(packet)
	pkts.Append(packet)

	for name, value := range initialParameterStatus {
		status = updateParameter(pkts, name, value)
		if status != Ok {
			return
		}
	}

	// send backend key data
	var cancellationKey [8]byte
	_, err := rand.Read(cancellationKey[:])
	if err != nil {
		fail(client, perror.Wrap(err))
		return
	}

	packet = zap.NewPacket()
	packets.WriteBackendKeyData(packet, cancellationKey)
	pkts.Append(packet)

	// send ready for query
	packet = zap.NewPacket()
	packets.WriteReadyForQuery(packet, 'I')
	pkts.Append(packet)

	err = client.WriteV(pkts)
	if err != nil {
		fail(client, perror.Wrap(err))
		return
	}

	ok = true
	return
}
