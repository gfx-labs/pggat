package frontends

import (
	"fmt"
	"strings"

	"pggat2/lib/perror"
	"pggat2/lib/util/slices"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
	"pggat2/lib/zap/packets/v3.0"
)

func startup0(
	conn zap.Conn,
	params *AcceptParams,
	options AcceptOptions,
) (done bool, err perror.Error) {
	packet, err2 := conn.ReadPacket(false)
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
			p.ReadBytes(params.CancelKey[:])

			if params.CancelKey == [8]byte{} {
				// very rare that this would ever happen
				// and it's ok if we don't honor cancel requests
				err = perror.New(
					perror.FATAL,
					perror.ProtocolViolation,
					"cancel key cannot be null",
				)
				return
			}

			done = true
			return
		case 5679:
			// ssl is not enabled
			if options.SSLConfig == nil {
				err = perror.Wrap(conn.WriteByte('N'))
				return
			}

			// do ssl
			if err = perror.Wrap(conn.WriteByte('S')); err != nil {
				return
			}
			if err = perror.Wrap(conn.EnableSSLServer(options.SSLConfig)); err != nil {
				return
			}
			params.SSLEnabled = true
			return
		case 5680:
			// GSSAPI is not supported yet
			err = perror.Wrap(conn.WriteByte('N'))
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
			params.User = value
		case "database":
			params.Database = value
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

					if params.InitialParameters == nil {
						params.InitialParameters = make(map[strutil.CIString]string)
					}
					params.InitialParameters[ikey] = value
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

				if params.InitialParameters == nil {
					params.InitialParameters = make(map[strutil.CIString]string)
				}
				params.InitialParameters[ikey] = value
			}
		}
	}

	if minorVersion != 0 || len(unsupportedOptions) > 0 {
		// negotiate protocol
		uopts := packets.NegotiateProtocolVersion{
			MinorProtocolVersion: 0,
			UnrecognizedOptions:  unsupportedOptions,
		}

		err = perror.Wrap(conn.WritePacket(uopts.IntoPacket()))
		if err != nil {
			return
		}
	}

	if params.User == "" {
		err = perror.New(
			perror.FATAL,
			perror.InvalidAuthorizationSpecification,
			"User is required",
		)
		return
	}
	if params.Database == "" {
		params.Database = params.User
	}

	done = true
	return
}

func accept(
	client zap.Conn,
	options AcceptOptions,
) (params AcceptParams, err perror.Error) {
	for {
		var done bool
		done, err = startup0(client, &params, options)
		if err != nil {
			return
		}
		if done {
			break
		}
	}

	if params.CancelKey != [8]byte{} {
		return
	}

	if options.SSLRequired && !params.SSLEnabled {
		err = perror.New(
			perror.FATAL,
			perror.InvalidPassword,
			"SSL is required",
		)
		return
	}

	return
}

func fail(client zap.Conn, err perror.Error) {
	resp := packets.ErrorResponse{
		Error: err,
	}
	_ = client.WritePacket(resp.IntoPacket())
}

func Accept(client zap.Conn, options AcceptOptions) (AcceptParams, perror.Error) {
	params, err := accept(client, options)
	if err != nil {
		fail(client, err)
		return AcceptParams{}, err
	}
	return params, nil
}
