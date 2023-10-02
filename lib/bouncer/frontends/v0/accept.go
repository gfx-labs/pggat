package frontends

import (
	"crypto/tls"
	"io"
	"strings"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func startup0(
	ctx *acceptContext,
	params *acceptParams,
) (cancelling bool, done bool, err perror.Error) {
	var err2 error
	ctx.Packet, err2 = ctx.Conn.ReadPacket(false, ctx.Packet)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}

	var majorVersion uint16
	var minorVersion uint16
	p := ctx.Packet.ReadUint16(&majorVersion)
	p = p.ReadUint16(&minorVersion)

	if majorVersion == 1234 {
		// Cancel or SSL
		switch minorVersion {
		case 5678:
			// Cancel
			p.ReadBytes(params.CancelKey[:])
			cancelling = true
			done = true
			return
		case 5679:
			byteWriter, ok := ctx.Conn.ReadWriteCloser.(io.ByteWriter)
			if !ok {
				err = perror.New(
					perror.FATAL,
					perror.FeatureNotSupported,
					"SSL is not supported",
				)
				return
			}

			// ssl is not enabled
			if ctx.Options.SSLConfig == nil {
				err = perror.Wrap(byteWriter.WriteByte('N'))
				return
			}

			sslServer, ok := ctx.Conn.ReadWriteCloser.(fed.SSLServer)
			if !ok {
				err = perror.Wrap(byteWriter.WriteByte('N'))
				return
			}

			// do ssl
			if err = perror.Wrap(byteWriter.WriteByte('S')); err != nil {
				return
			}
			if err = perror.Wrap(sslServer.EnableSSLServer(ctx.Options.SSLConfig)); err != nil {
				return
			}
			return
		case 5680:
			byteWriter, ok := ctx.Conn.ReadWriteCloser.(io.ByteWriter)
			if !ok {
				err = perror.New(
					perror.FATAL,
					perror.FeatureNotSupported,
					"GSSAPI is not supported",
				)
				return
			}

			// GSSAPI is not supported yet
			err = perror.Wrap(byteWriter.WriteByte('N'))
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
			ctx.Conn.User = value
		case "database":
			ctx.Conn.Database = value
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

					if ctx.Conn.InitialParameters == nil {
						ctx.Conn.InitialParameters = make(map[strutil.CIString]string)
					}
					ctx.Conn.InitialParameters[ikey] = value
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

				if ctx.Conn.InitialParameters == nil {
					ctx.Conn.InitialParameters = make(map[strutil.CIString]string)
				}
				ctx.Conn.InitialParameters[ikey] = value
			}
		}
	}

	if minorVersion != 0 || len(unsupportedOptions) > 0 {
		// negotiate protocol
		uopts := packets.NegotiateProtocolVersion{
			MinorProtocolVersion: 0,
			UnrecognizedOptions:  unsupportedOptions,
		}
		ctx.Packet = uopts.IntoPacket(ctx.Packet)
		err = perror.Wrap(ctx.Conn.WritePacket(ctx.Packet))
		if err != nil {
			return
		}
	}

	if ctx.Conn.User == "" {
		err = perror.New(
			perror.FATAL,
			perror.InvalidAuthorizationSpecification,
			"User is required",
		)
		return
	}
	if ctx.Conn.Database == "" {
		ctx.Conn.Database = ctx.Conn.User
	}

	done = true
	return
}

func accept0(
	ctx *acceptContext,
) (params acceptParams, err perror.Error) {
	for {
		var done bool
		params.IsCanceling, done, err = startup0(ctx, &params)
		if err != nil {
			return
		}
		if done {
			break
		}
	}

	return
}

func fail(packet fed.Packet, client fed.ReadWriter, err perror.Error) {
	resp := packets.ErrorResponse{
		Error: err,
	}
	packet = resp.IntoPacket(packet)
	_ = client.WritePacket(packet)
}

func accept(ctx *acceptContext) (acceptParams, perror.Error) {
	params, err := accept0(ctx)
	if err != nil {
		fail(ctx.Packet, ctx.Conn, err)
		return acceptParams{}, err
	}
	return params, nil
}

func Accept(conn *fed.Conn, tlsConfig *tls.Config) (
	cancelKey [8]byte,
	isCanceling bool,
	err perror.Error,
) {
	ctx := acceptContext{
		Conn: conn,
		Options: acceptOptions{
			SSLConfig: tlsConfig,
		},
	}
	var params acceptParams
	params, err = accept(&ctx)
	cancelKey = params.CancelKey
	isCanceling = params.IsCanceling
	return
}
