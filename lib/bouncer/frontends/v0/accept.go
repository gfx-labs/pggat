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
	packet, err2 := client.RW.ReadPacket(false)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}

	var majorVersion uint16
	var minorVersion uint16
	p := packet.ReadUint16(&majorVersion)
	p = p.ReadUint16(&minorVersion)

	if majorVersion == 1234 {
		// Cancel or SSL
		switch minorVersion {
		case 5678:
			// Cancel
			p.ReadBytes(client.BackendKey[:])

			options.Pooler.Cancel(client.BackendKey)

			err = perror.New(
				perror.FATAL,
				perror.ProtocolViolation,
				"Expected client to disconnect",
			)
			return
		case 5679:
			// ssl is not enabled
			if options.SSLConfig == nil {
				err = perror.Wrap(client.RW.WriteByte('N'))
				return
			}

			// do ssl
			if err = perror.Wrap(client.RW.WriteByte('S')); err != nil {
				return
			}
			if err = perror.Wrap(client.RW.EnableSSLServer(options.SSLConfig)); err != nil {
				return
			}
			client.SSLEnabled = true
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
		var key string
		p = p.ReadString(&key)
		if key == "" {
			break
		}

		var value string
		p = p.ReadString(&value)

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
					var ok bool
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
		uopts := packets.NegotiateProtocolVersion{
			MinorProtocolVersion: 0,
			UnrecognizedOptions:  unsupportedOptions,
		}

		err = perror.Wrap(client.RW.WritePacket(uopts.IntoPacket()))
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
	packet, err2 := client.ReadPacket(true)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}
	var initialResponse packets.SASLInitialResponse
	if !initialResponse.ReadFromPacket(packet) {
		err = packets.ErrBadFormat
		return
	}

	tool, err2 = creds.VerifySASL(initialResponse.Mechanism)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}

	resp, err2 = tool.Write(initialResponse.InitialResponse)
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
	packet, err2 := client.ReadPacket(true)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}
	var authResp packets.AuthenticationResponse
	if !authResp.ReadFromPacket(packet) {
		err = packets.ErrBadFormat
		return
	}

	resp, err2 = tool.Write(authResp)
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
	saslInitial := packets.AuthenticationSASL{
		Mechanisms: creds.SupportedSASLMechanisms(),
	}
	err := perror.Wrap(client.WritePacket(saslInitial.IntoPacket()))
	if err != nil {
		return err
	}

	tool, resp, done, err := authenticationSASLInitial(client, creds)
	if err != nil {
		return err
	}

	for {
		if done {
			final := packets.AuthenticationSASLFinal(resp)
			err = perror.Wrap(client.WritePacket(final.IntoPacket()))
			if err != nil {
				return err
			}
			break
		} else {
			cont := packets.AuthenticationSASLContinue(resp)
			err = perror.Wrap(client.WritePacket(cont.IntoPacket()))
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

func updateParameter(client zap.ReadWriter, name, value string) perror.Error {
	ps := packets.ParameterStatus{
		Key:   name,
		Value: value,
	}
	return perror.Wrap(client.WritePacket(ps.IntoPacket()))
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

	if options.SSLRequired && !conn.SSLEnabled {
		err = perror.New(
			perror.FATAL,
			perror.InvalidPassword,
			"SSL is required",
		)
		return
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

	// send auth Ok
	authOk := packets.AuthenticationOk{}
	if err = perror.Wrap(client.WritePacket(authOk.IntoPacket())); err != nil {
		return
	}

	// send backend key data
	_, err2 := rand.Read(conn.BackendKey[:])
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}

	keyData := packets.BackendKeyData{
		CancellationKey: conn.BackendKey,
	}
	if err = perror.Wrap(client.WritePacket(keyData.IntoPacket())); err != nil {
		return
	}

	if err = updateParameter(client, "client_encoding", "UTF8"); err != nil {
		return
	}
	if err = updateParameter(client, "server_encoding", "UTF8"); err != nil {
		return
	}
	if err = updateParameter(client, "server_version", "14.5"); err != nil {
		return
	}

	// send ready for query
	rfq := packets.ReadyForQuery('I')
	if err = perror.Wrap(client.WritePacket(rfq.IntoPacket())); err != nil {
		return
	}

	return
}

func fail(client zap.ReadWriter, err perror.Error) {
	resp := packets.ErrorResponse{
		Error: err,
	}
	_ = client.WritePacket(resp.IntoPacket())
}

func Accept(client zap.ReadWriter, options AcceptOptions) (bouncer.Conn, perror.Error) {
	conn, err := accept(client, options)
	if err != nil {
		fail(client, err)
		return bouncer.Conn{}, err
	}
	return conn, nil
}
