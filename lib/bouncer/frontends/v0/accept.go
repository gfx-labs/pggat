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

func startup0(client zap.ReadWriter) (done bool, status Status) {
	packet := zap.NewUntypedPacket()
	defer packet.Done()
	err := client.ReadUntyped(packet)
	if err != nil {
		fail(client, perror.Wrap(err))
		return false, Fail
	}
	read := packet.Read()

	majorVersion, ok := read.ReadUint16()
	if !ok {
		fail(client, packets.ErrBadFormat)
		return false, Fail
	}
	minorVersion, ok := read.ReadUint16()
	if !ok {
		fail(client, packets.ErrBadFormat)
		return false, Fail
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
			return false, Fail
		case 5679:
			// SSL is not supported yet
			err = client.WriteByte('N')
			if err != nil {
				fail(client, perror.Wrap(err))
				return false, Fail
			}
			return false, Ok
		case 5680:
			// GSSAPI is not supported yet
			err = client.WriteByte('N')
			if err != nil {
				fail(client, perror.Wrap(err))
				return false, Fail
			}
			return false, Ok
		default:
			fail(client, perror.New(
				perror.FATAL,
				perror.ProtocolViolation,
				"Unknown request code",
			))
			return false, Fail
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

	var user string
	var database string

	for {
		key, ok := read.ReadString()
		if !ok {
			fail(client, packets.ErrBadFormat)
			return false, Fail
		}
		if key == "" {
			break
		}

		value, ok := read.ReadString()
		if !ok {
			fail(client, packets.ErrBadFormat)
			return false, Fail
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
			return false, Fail
		case "replication":
			fail(client, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"Replication mode is not supported yet",
			))
			return false, Fail
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
			return false, Fail
		}
	}

	if user == "" {
		fail(client, perror.New(
			perror.FATAL,
			perror.InvalidAuthorizationSpecification,
			"User is required",
		))
		return false, Fail
	}
	if database == "" {
		database = user
	}

	return true, Ok
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

func updateParameter(client zap.ReadWriter, name, value string) Status {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteParameterStatus(packet, name, value)
	err := client.Write(packet)
	if err != nil {
		fail(client, perror.Wrap(err))
		return Fail
	}
	return Ok
}

func Accept(client zap.ReadWriter, initialParameterStatus map[string]string) {
	for {
		done, status := startup0(client)
		if status != Ok {
			return
		}
		if done {
			break
		}
	}

	status := authenticationSASL(client, "test", "pw")
	if status != Ok {
		return
	}

	// send auth Ok
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteAuthenticationOk(packet)
	err := client.Write(packet)
	if err != nil {
		fail(client, perror.Wrap(err))
		return
	}

	for name, value := range initialParameterStatus {
		status = updateParameter(client, name, value)
		if status != Ok {
			return
		}
	}

	// send backend key data
	var cancellationKey [8]byte
	_, err = rand.Read(cancellationKey[:])
	if err != nil {
		fail(client, perror.Wrap(err))
		return
	}
	packets.WriteBackendKeyData(packet, cancellationKey)
	err = client.Write(packet)
	if err != nil {
		fail(client, perror.Wrap(err))
		return
	}

	// send ready for query
	packets.WriteReadyForQuery(packet, 'I')
	err = client.Write(packet)
	if err != nil {
		fail(client, perror.Wrap(err))
		return
	}
}
