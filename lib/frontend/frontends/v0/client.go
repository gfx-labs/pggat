package frontends

import (
	"crypto/rand"
	"net"

	"pggat2/lib/auth"
	"pggat2/lib/frontend"
	"pggat2/lib/perror"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
)

var ErrBadPacketFormat = perror.New(
	perror.FATAL,
	perror.ProtocolViolation,
	"Bad packet format",
)

type Client struct {
	conn net.Conn

	pnet.Reader
	pnet.Writer

	user     string
	database string

	// cancellation key data
	cancellationKey [8]byte
}

func NewClient(conn net.Conn) (*Client, error) {
	client := &Client{
		conn:   conn,
		Reader: pnet.MakeReader(conn),
		Writer: pnet.MakeWriter(conn),
	}
	err := client.accept()
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (T *Client) accept() error {
	for {
		startup, err := T.ReadUntyped()
		if err != nil {
			return err
		}
		reader := packet.MakeReader(startup)

		majorVersion, ok := reader.Uint16()
		if !ok {
			return T.Error(ErrBadPacketFormat)
		}
		minorVersion, ok := reader.Uint16()
		if !ok {
			return T.Error(ErrBadPacketFormat)
		}

		if majorVersion == 1234 {
			// Cancel or SSL
			switch minorVersion {
			case 5678:
				// Cancel
				err = T.Error(perror.New(
					perror.FATAL,
					perror.FeatureNotSupported,
					"Cancel is not supported yet",
				))
				if err != nil {
					return err
				}
				continue
			case 5679:
				// SSL is not supported yet
				err = T.WriteByte('N')
				if err != nil {
					return err
				}
				continue
			case 5680:
				// GSSAPI is not supported yet
				err = T.WriteByte('N')
				if err != nil {
					return err
				}
				continue
			default:
				err = T.Error(perror.New(
					perror.FATAL,
					perror.ProtocolViolation,
					"Unknown request code",
				))
				if err != nil {
					return err
				}
				continue
			}
		}

		if majorVersion != 3 {
			err = T.Error(perror.New(
				perror.FATAL,
				perror.ProtocolViolation,
				"Unsupported protocol version",
			))
		}

		var unsupportedOptions []string

		for {
			key, ok := reader.String()
			if !ok {
				return T.Error(ErrBadPacketFormat)
			}
			if key == "" {
				break
			}

			value, ok := reader.String()
			if !ok {
				return T.Error(ErrBadPacketFormat)
			}

			switch key {
			case "user":
				T.user = value
			case "database":
				T.database = value
			case "options":
				return T.Error(perror.New(
					perror.FATAL,
					perror.FeatureNotSupported,
					"Startup options are not supported yet",
				))
			case "replication":
				return T.Error(perror.New(
					perror.FATAL,
					perror.FeatureNotSupported,
					"Replication mode is not supported yet",
				))
			default:
				unsupportedOptions = append(unsupportedOptions, key)
			}
		}

		if minorVersion != 0 || len(unsupportedOptions) > 0 {
			// negotiate protocol
			var builder packet.Builder
			builder.Type(packet.NegotiateProtocolVersion)
			builder.Int32(0)
			builder.Int32(int32(len(unsupportedOptions)))
			for _, v := range unsupportedOptions {
				builder.String(v)
			}

			err = T.Write(builder.Raw())
			if err != nil {
				return err
			}
		}

		if T.user == "" {
			return T.Error(perror.New(
				perror.FATAL,
				perror.InvalidAuthorizationSpecification,
				"User is required",
			))
		}
		if T.database == "" {
			T.database = T.user
		}

		break
	}

	var salt [4]byte
	_, err := rand.Read(salt[:])
	if err != nil {
		return err
	}

	// password time
	// build cleartext password packet
	var builder packet.Builder
	builder.Type(packet.Authentication)
	builder.Uint32(5)
	builder.Bytes(salt[:])

	err = T.Write(builder.Raw())
	if err != nil {
		return err
	}

	// read password
	password, err := T.Read()
	if err != nil {
		return err
	}

	reader := packet.MakeReader(password)
	if reader.Type() != packet.AuthenticationResponse {
		return T.Error(perror.New(
			perror.FATAL,
			perror.ProtocolViolation,
			"Expected password",
		))
	}

	pw, ok := reader.String()
	if !ok {
		return T.Error(ErrBadPacketFormat)
	}

	if !auth.CheckMD5("test", "password", salt, pw) {
		return T.Error(perror.New(
			perror.FATAL,
			perror.InvalidPassword,
			"Invalid password",
		))
	}

	// send auth ok
	builder = packet.Builder{}
	builder.Type(packet.Authentication)
	builder.Uint32(0)

	err = T.Write(builder.Raw())
	if err != nil {
		return err
	}

	// send backend key data
	_, err = rand.Read(T.cancellationKey[:])
	if err != nil {
		return err
	}
	builder = packet.Builder{}
	builder.Type(packet.BackendKeyData)
	builder.Bytes(T.cancellationKey[:])

	err = T.Write(builder.Raw())
	if err != nil {
		return err
	}

	// send ready for query
	builder = packet.Builder{}
	builder.Type(packet.ReadyForQuery)
	builder.Uint8('I')

	err = T.Write(builder.Raw())
	if err != nil {
		return err
	}

	return nil
}

func (T *Client) Error(err perror.Error) error {
	var builder packet.Builder
	builder.Type(packet.ErrorResponse)

	builder.Uint8('S')
	builder.String(string(err.Severity()))

	builder.Uint8('C')
	builder.String(string(err.Code()))

	builder.Uint8('M')
	builder.String(err.Message())

	for _, field := range err.Extra() {
		builder.Uint8(uint8(field.Type))
		builder.String(field.Value)
	}

	builder.Uint8(0)

	return T.Write(builder.Raw())
}

var _ frontend.Client = (*Client)(nil)
