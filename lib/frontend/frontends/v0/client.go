package frontends

import (
	"crypto/rand"
	"net"

	"pggat2/lib/auth/md5"
	"pggat2/lib/auth/sasl"
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

var ErrProtocolError = perror.New(
	perror.FATAL,
	perror.ProtocolViolation,
	"Expected a different packet",
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

func (T *Client) startup0() (bool, error) {
	startup, err := T.ReadUntyped()
	if err != nil {
		return false, err
	}
	reader := packet.MakeReader(startup)

	majorVersion, ok := reader.Uint16()
	if !ok {
		return false, T.Error(ErrBadPacketFormat)
	}
	minorVersion, ok := reader.Uint16()
	if !ok {
		return false, T.Error(ErrBadPacketFormat)
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
			return false, err
		case 5679:
			// SSL is not supported yet
			err = T.WriteByte('N')
			return false, err
		case 5680:
			// GSSAPI is not supported yet
			err = T.WriteByte('N')
			return false, err
		default:
			err = T.Error(perror.New(
				perror.FATAL,
				perror.ProtocolViolation,
				"Unknown request code",
			))
			return false, err
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
			return false, T.Error(ErrBadPacketFormat)
		}
		if key == "" {
			break
		}

		value, ok := reader.String()
		if !ok {
			return false, T.Error(ErrBadPacketFormat)
		}

		switch key {
		case "user":
			T.user = value
		case "database":
			T.database = value
		case "options":
			return false, T.Error(perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"Startup options are not supported yet",
			))
		case "replication":
			return false, T.Error(perror.New(
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
			return false, err
		}
	}

	if T.user == "" {
		return false, T.Error(perror.New(
			perror.FATAL,
			perror.InvalidAuthorizationSpecification,
			"User is required",
		))
	}
	if T.database == "" {
		T.database = T.user
	}

	return true, nil
}

func (T *Client) authenticationSASL(username, password string) error {
	var builder packet.Builder
	builder.Type(packet.Authentication)
	builder.Int32(10)
	for _, mechanism := range sasl.Mechanisms {
		builder.String(mechanism)
	}
	builder.String("")

	err := T.Write(builder.Raw())
	if err != nil {
		return err
	}

	// check which authentication method the client wants
	pkt, err := T.Read()
	if err != nil {
		return err
	}
	if pkt.Type != packet.AuthenticationResponse {
		return T.Error(ErrBadPacketFormat)
	}

	reader := packet.MakeReader(pkt)
	mechanism, ok := reader.String()
	if !ok {
		return T.Error(ErrBadPacketFormat)
	}
	tool, err := sasl.NewServer(mechanism, username, password)
	if err != nil {
		return err
	}
	_, ok = reader.Int32()
	if !ok {
		return T.Error(ErrBadPacketFormat)
	}

	resp, done, err := tool.InitialResponse(reader.Remaining())

	for {
		if err != nil {
			return err
		}
		if done {
			builder = packet.Builder{}
			builder.Type(packet.Authentication)
			builder.Int32(12)
			builder.Bytes(resp)
			err = T.Write(builder.Raw())
			if err != nil {
				return err
			}
			break
		} else {
			builder = packet.Builder{}
			builder.Type(packet.Authentication)
			builder.Int32(11)
			builder.Bytes(resp)
			err = T.Write(builder.Raw())
			if err != nil {
				return err
			}
		}

		pkt, err = T.Read()
		if err != nil {
			return err
		}
		if pkt.Type != packet.AuthenticationResponse {
			return T.Error(ErrProtocolError)
		}

		resp, done, err = tool.Continue(pkt.Payload)
	}

	return nil
}

func (T *Client) authenticationMD5(username, password string) error {
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
	pkt, err := T.Read()
	if err != nil {
		return err
	}

	reader := packet.MakeReader(pkt)
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

	if !md5.Check(username, password, salt, pw) {
		return T.Error(perror.New(
			perror.FATAL,
			perror.InvalidPassword,
			"Invalid password",
		))
	}

	return nil
}

func (T *Client) accept() error {
	for {
		done, err := T.startup0()
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	// TODO(garet) don't hardcode username and password
	err := T.authenticationSASL("test", "password")
	if err != nil {
		return err
	}

	// send auth ok
	builder := packet.Builder{}
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
