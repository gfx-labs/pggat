package frontends

import (
	"crypto/tls"
	"strings"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func startup0(
	ctx *acceptContext,
	params *acceptParams,
) (cancelling bool, done bool, err error) {
	var packet fed.Packet
	packet, err = ctx.Conn.ReadPacket(false)
	if err != nil {
		return
	}

	var p packets.Startup
	err = fed.ToConcrete(&p, packet)
	if err != nil {
		return
	}

	switch mode := p.Mode.(type) {
	case *packets.StartupPayloadControl:
		switch control := mode.Mode.(type) {
		case *packets.StartupPayloadControlPayloadCancel:
			// Cancel
			params.CancelKey.ProcessID = control.ProcessID
			params.CancelKey.SecretKey = control.SecretKey
			cancelling = true
			done = true
			return
		case *packets.StartupPayloadControlPayloadSSL:
			// ssl is not enabled
			if ctx.Options.SSLConfig == nil {
				err = ctx.Conn.WriteByte('N')
				return
			}

			// do ssl
			if err = ctx.Conn.WriteByte('S'); err != nil {
				return
			}
			if err = ctx.Conn.EnableSSL(ctx.Options.SSLConfig, false); err != nil {
				return
			}
			return
		case *packets.StartupPayloadControlPayloadGSSAPI:
			// GSSAPI is not supported yet
			err = ctx.Conn.WriteByte('N')
			return
		default:
			err = perror.New(
				perror.FATAL,
				perror.ProtocolViolation,
				"Unknown request code",
			)
			return
		}
	case *packets.StartupPayloadVersion3:
		var unsupportedOptions []string

		for _, parameter := range mode.Parameters {
			switch parameter.Key {
			case "user":
				ctx.Conn.User = parameter.Value
			case "database":
				ctx.Conn.Database = parameter.Value
			case "options":
				fields := strings.Fields(parameter.Value)
				for i := 0; i < len(fields); i++ {
					switch fields[i] {
					case "-c":
						i++
						set := fields[i]
						key, value, ok := strings.Cut(set, "=")
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
				if strings.HasPrefix(parameter.Key, "_pq_.") {
					// we don't support protocol extensions at the moment
					unsupportedOptions = append(unsupportedOptions, parameter.Key)
				} else {
					ikey := strutil.MakeCIString(parameter.Key)

					if ctx.Conn.InitialParameters == nil {
						ctx.Conn.InitialParameters = make(map[strutil.CIString]string)
					}
					ctx.Conn.InitialParameters[ikey] = parameter.Value
				}
			}
		}

		if mode.MinorVersion != 0 || len(unsupportedOptions) > 0 {
			// negotiate protocol
			uopts := packets.NegotiateProtocolVersion{
				MinorProtocolVersion:        0,
				UnrecognizedProtocolOptions: unsupportedOptions,
			}
			err = ctx.Conn.WritePacket(&uopts)
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
	default:
		err = perror.New(
			perror.FATAL,
			perror.ProtocolViolation,
			"Unsupported protocol version",
		)
		return
	}
}

func accept0(
	ctx *acceptContext,
) (params acceptParams, err error) {
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

func fail(client *fed.Conn, err error) {
	resp := perror.ToPacket(perror.Wrap(err))
	_ = client.WritePacket(resp)
}

func accept(ctx *acceptContext) (acceptParams, error) {
	params, err := accept0(ctx)
	if err != nil {
		fail(ctx.Conn, err)
		return acceptParams{}, err
	}
	return params, nil
}

func Accept(conn *fed.Conn, tlsConfig *tls.Config) (
	cancelKey fed.BackendKey,
	isCanceling bool,
	err error,
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
