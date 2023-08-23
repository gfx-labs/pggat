package frontends

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"pggat2/lib/auth"
	"pggat2/lib/bouncer"
	"pggat2/lib/perror"
	"pggat2/lib/util/slices"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
	"pggat2/lib/zap/packets/v3.0"
)

func startup0(
	client *bouncer.Conn,
	options AcceptOptions,
) (done bool, err perror.Error) {
	packet := zap.NewUntypedPacket()
	defer packet.Done()
	err = perror.Wrap(client.RW.ReadUntyped(packet))
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
			if !read.ReadBytes(client.BackendKey[:]) {
				err = packets.ErrBadFormat
				return
			}

			options.Pooler.Cancel(client.BackendKey)

			err = perror.New(
				perror.FATAL,
				perror.ProtocolViolation,
				"Expected client to disconnect",
			)
			return
		case 5679:
			// SSL is not supported yet
			if err = perror.Wrap(client.RW.WriteByte('S')); err != nil {
				return
			}
			if err = perror.Wrap(client.RW.EnableSSL(false)); err != nil {
				return
			}
			return
		case 5680:
			// GSSAPI is not supported yet
			err = perror.Wrap(client.RW.WriteByte('N'))
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
			client.User = value
		case "database":
			client.Database = value
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

					ikey := strutil.MakeCIString(key)

					if !slices.Contains(options.AllowedStartupOptions, ikey) {
						err = perror.New(
							perror.FATAL,
							perror.FeatureNotSupported,
							fmt.Sprintf(`Startup parameter "%s" is not allowed`, key),
						)
						return
					}

					if client.InitialParameters == nil {
						client.InitialParameters = make(map[strutil.CIString]string)
					}
					client.InitialParameters[ikey] = value
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
				ikey := strutil.MakeCIString(key)

				if !slices.Contains(options.AllowedStartupOptions, ikey) {
					err = perror.New(
						perror.FATAL,
						perror.FeatureNotSupported,
						fmt.Sprintf(`Startup parameter "%s" is not allowed`, key),
					)
					return
				}

				if client.InitialParameters == nil {
					client.InitialParameters = make(map[strutil.CIString]string)
				}
				client.InitialParameters[ikey] = value
			}
		}
	}

	if minorVersion != 0 || len(unsupportedOptions) > 0 {
		// negotiate protocol
		packet := zap.NewPacket()
		defer packet.Done()
		packets.WriteNegotiateProtocolVersion(packet, 0, unsupportedOptions)

		err = perror.Wrap(client.RW.Write(packet))
		if err != nil {
			return
		}
	}

	if client.User == "" {
		err = perror.New(
			perror.FATAL,
			perror.InvalidAuthorizationSpecification,
			"User is required",
		)
		return
	}
	if client.Database == "" {
		client.Database = client.User
	}

	done = true
	return
}

func authenticationSASLInitial(client zap.ReadWriter, creds auth.SASL) (tool auth.SASLVerifier, resp []byte, done bool, err perror.Error) {
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
	tool, err2 = creds.VerifySASL(mechanism)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}

	resp, err2 = tool.Write(initialResponse)
	if err2 != nil {
		if errors.Is(err2, auth.ErrSASLComplete) {
			done = true
			return
		}
		err = perror.Wrap(err2)
		return
	}
	return
}

func authenticationSASLContinue(client zap.ReadWriter, tool auth.SASLVerifier) (resp []byte, done bool, err perror.Error) {
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
	resp, err2 = tool.Write(clientResp)
	if err2 != nil {
		if errors.Is(err2, auth.ErrSASLComplete) {
			done = true
			return
		}
		err = perror.Wrap(err2)
		return
	}
	return
}

func authenticationSASL(client zap.ReadWriter, creds auth.SASL) perror.Error {
	packet := zap.NewPacket()
	defer packet.Done()
	packets.WriteAuthenticationSASL(packet, creds.SupportedSASLMechanisms())
	err := perror.Wrap(client.Write(packet))
	if err != nil {
		return err
	}

	tool, resp, done, err := authenticationSASLInitial(client, creds)
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

func accept(
	client zap.ReadWriter,
	options AcceptOptions,
) (conn bouncer.Conn, err perror.Error) {
	conn.RW = client

	for {
		var done bool
		done, err = startup0(&conn, options)
		if err != nil {
			return
		}
		if done {
			break
		}
	}

	creds := options.Pooler.GetUserCredentials(conn.User, conn.Database)
	if creds == nil {
		err = perror.New(
			perror.FATAL,
			perror.InvalidPassword,
			"User or database not found",
		)
		return
	}
	if credsSASL, ok := creds.(auth.SASL); ok {
		err = authenticationSASL(client, credsSASL)
	} else {
		err = perror.New(
			perror.FATAL,
			perror.InternalError,
			"Auth method not supported",
		)
	}
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
	_, err2 := rand.Read(conn.BackendKey[:])
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}

	packet = zap.NewPacket()
	packets.WriteBackendKeyData(packet, conn.BackendKey)
	pkts.Append(packet)

	if conn.InitialParameters == nil {
		conn.InitialParameters = make(map[strutil.CIString]string)
	}
	conn.InitialParameters[strutil.MakeCIString("client_encoding")] = "UTF8"
	conn.InitialParameters[strutil.MakeCIString("server_encoding")] = "UTF8"
	conn.InitialParameters[strutil.MakeCIString("server_version")] = "14.5"
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

func Accept(client zap.ReadWriter, options AcceptOptions) (bouncer.Conn, perror.Error) {
	conn, err := accept(client, options)
	if err != nil {
		fail(client, err)
		return bouncer.Conn{}, err
	}
	return conn, nil
}
