package frontends

import (
	"context"
	"crypto/tls"
	"strings"

	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type acceptParams struct {
	Conn    *fed.Conn
	Options acceptOptions
}

type acceptResult struct {
	CancelKey   fed.BackendKey
	IsCanceling bool
}

func startup0(
	ctx context.Context,
	params *acceptParams,
	result *acceptResult,
) (cancelling bool, done bool, err error) {
	var packet fed.Packet
	packet, err = params.Conn.ReadPacket(ctx, false)
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
			result.CancelKey.ProcessID = control.ProcessID
			result.CancelKey.SecretKey = control.SecretKey
			cancelling = true
			done = true
			return
		case *packets.StartupPayloadControlPayloadSSL:
			// ssl is not enabled
			if params.Options.SSLConfig == nil {
				err = params.Conn.WriteByte(ctx, 'N')
				return
			}

			// do ssl
			if err = params.Conn.WriteByte(ctx, 'S'); err != nil {
				return
			}
			if err = params.Conn.EnableSSL(ctx, params.Options.SSLConfig, false); err != nil {
				return
			}
			return
		case *packets.StartupPayloadControlPayloadGSSAPI:
			// GSSAPI is not supported yet
			err = params.Conn.WriteByte(ctx, 'N')
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
				params.Conn.User = parameter.Value
			case "database":
				params.Conn.Database = parameter.Value
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

						if params.Conn.InitialParameters == nil {
							params.Conn.InitialParameters = make(map[strutil.CIString]string)
						}
						params.Conn.InitialParameters[ikey] = value
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

					if params.Conn.InitialParameters == nil {
						params.Conn.InitialParameters = make(map[strutil.CIString]string)
					}
					params.Conn.InitialParameters[ikey] = parameter.Value
				}
			}
		}

		if mode.MinorVersion != 0 || len(unsupportedOptions) > 0 {
			// negotiate protocol
			uopts := packets.NegotiateProtocolVersion{
				MinorProtocolVersion:        0,
				UnrecognizedProtocolOptions: unsupportedOptions,
			}
			err = params.Conn.WritePacket(ctx, &uopts)
			if err != nil {
				return
			}
		}

		if params.Conn.User == "" {
			err = perror.New(
				perror.FATAL,
				perror.InvalidAuthorizationSpecification,
				"User is required",
			)
			return
		}
		if params.Conn.Database == "" {
			params.Conn.Database = params.Conn.User
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
	ctx context.Context,
	params *acceptParams,
) (result acceptResult, err error) {
	for {
		var done bool
		result.IsCanceling, done, err = startup0(ctx, params, &result)
		if err != nil {
			return
		}
		if done {
			break
		}
	}

	return
}

func fail(ctx context.Context, client *fed.Conn, err error) {
	resp := perror.ToPacket(perror.Wrap(err))
	_ = client.WritePacket(ctx, resp)
}

func accept(ctx context.Context, params *acceptParams) (acceptResult, error) {
	result, err := accept0(ctx, params)
	if err != nil {
		fail(ctx, params.Conn, err)
		return acceptResult{}, err
	}
	return result, nil
}

func Accept(conn *fed.Conn, tlsConfig *tls.Config) (
	cancelKey fed.BackendKey,
	isCanceling bool,
	err error,
) {
	params := acceptParams{
		Conn: conn,
		Options: acceptOptions{
			SSLConfig: tlsConfig,
		},
	}
	var result acceptResult
	if result, err = accept(context.Background(), &params); err == nil {
		cancelKey = result.CancelKey
		isCanceling = result.IsCanceling
	}
	return
}
