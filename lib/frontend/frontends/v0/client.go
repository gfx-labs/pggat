package frontends

import (
	"crypto/rand"
	"net"
	"strings"

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

func WrapError(err error) perror.Error {
	if err == nil {
		return nil
	}
	return perror.New(
		perror.FATAL,
		perror.InternalError,
		err.Error(),
	)
}

type Client struct {
	conn net.Conn

	pnet.Reader
	pnet.Writer

	user     string
	database string

	// cancellation key data
	cancellationKey [8]byte
	parameters      map[string]string
}

func NewClient(conn net.Conn) *Client {
	client := &Client{
		conn:       conn,
		Reader:     pnet.MakeReader(conn),
		Writer:     pnet.MakeWriter(conn),
		parameters: make(map[string]string),
	}
	err := client.accept()
	if err != nil {
		client.Close(err)
		return nil
	}
	return client
}

func (T *Client) startup0() (bool, perror.Error) {
	startup, err := T.ReadUntyped()
	if err != nil {
		return false, WrapError(err)
	}
	reader := packet.MakeReader(startup)

	majorVersion, ok := reader.Uint16()
	if !ok {
		return false, ErrBadPacketFormat
	}
	minorVersion, ok := reader.Uint16()
	if !ok {
		return false, ErrBadPacketFormat
	}

	if majorVersion == 1234 {
		// Cancel or SSL
		switch minorVersion {
		case 5678:
			// Cancel
			return false, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"Cancel is not supported yet",
			)
		case 5679:
			// SSL is not supported yet
			err = T.WriteByte('N')
			return false, WrapError(err)
		case 5680:
			// GSSAPI is not supported yet
			err = T.WriteByte('N')
			return false, WrapError(err)
		default:
			return false, perror.New(
				perror.FATAL,
				perror.ProtocolViolation,
				"Unknown request code",
			)
		}
	}

	if majorVersion != 3 {
		return false, perror.New(
			perror.FATAL,
			perror.ProtocolViolation,
			"Unsupported protocol version",
		)
	}

	var unsupportedOptions []string

	for {
		key, ok := reader.String()
		if !ok {
			return false, ErrBadPacketFormat
		}
		if key == "" {
			break
		}

		value, ok := reader.String()
		if !ok {
			return false, ErrBadPacketFormat
		}

		switch key {
		case "user":
			T.user = value
		case "database":
			T.database = value
		case "options":
			return false, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"Startup options are not supported yet",
			)
		case "replication":
			return false, perror.New(
				perror.FATAL,
				perror.FeatureNotSupported,
				"Replication mode is not supported yet",
			)
		default:
			if strings.HasPrefix(key, "_pq_.") {
				// we don't support protocol extensions at the moment
				unsupportedOptions = append(unsupportedOptions, key)
			} else {
				T.parameters[key] = value
			}
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
			return false, WrapError(err)
		}
	}

	if T.user == "" {
		return false, perror.New(
			perror.FATAL,
			perror.InvalidAuthorizationSpecification,
			"User is required",
		)
	}
	if T.database == "" {
		T.database = T.user
	}

	return true, nil
}

func (T *Client) authenticationSASL(username, password string) perror.Error {
	var builder packet.Builder
	builder.Type(packet.Authentication)
	builder.Int32(10)
	for _, mechanism := range sasl.Mechanisms {
		builder.String(mechanism)
	}
	builder.String("")

	err := T.Write(builder.Raw())
	if err != nil {
		return WrapError(err)
	}

	// check which authentication method the client wants
	pkt, err := T.Read()
	if err != nil {
		return WrapError(err)
	}
	if pkt.Type != packet.AuthenticationResponse {
		return ErrBadPacketFormat
	}

	reader := packet.MakeReader(pkt)
	mechanism, ok := reader.String()
	if !ok {
		return ErrBadPacketFormat
	}
	tool, err := sasl.NewServer(mechanism, username, password)
	if err != nil {
		return WrapError(err)
	}
	_, ok = reader.Int32()
	if !ok {
		return ErrBadPacketFormat
	}

	resp, done, err := tool.InitialResponse(reader.Remaining())

	for {
		if err != nil {
			return WrapError(err)
		}
		if done {
			builder = packet.Builder{}
			builder.Type(packet.Authentication)
			builder.Int32(12)
			builder.Bytes(resp)
			err = T.Write(builder.Raw())
			if err != nil {
				return WrapError(err)
			}
			break
		} else {
			builder = packet.Builder{}
			builder.Type(packet.Authentication)
			builder.Int32(11)
			builder.Bytes(resp)
			err = T.Write(builder.Raw())
			if err != nil {
				return WrapError(err)
			}
		}

		pkt, err = T.Read()
		if err != nil {
			return WrapError(err)
		}
		if pkt.Type != packet.AuthenticationResponse {
			return ErrProtocolError
		}

		resp, done, err = tool.Continue(pkt.Payload)
	}

	return nil
}

func (T *Client) authenticationMD5(username, password string) perror.Error {
	var salt [4]byte
	_, err := rand.Read(salt[:])
	if err != nil {
		return WrapError(err)
	}

	// password time
	// build cleartext password packet
	var builder packet.Builder
	builder.Type(packet.Authentication)
	builder.Uint32(5)
	builder.Bytes(salt[:])

	err = T.Write(builder.Raw())
	if err != nil {
		return WrapError(err)
	}

	// read password
	pkt, err := T.Read()
	if err != nil {
		return WrapError(err)
	}

	reader := packet.MakeReader(pkt)
	if reader.Type() != packet.AuthenticationResponse {
		return perror.New(
			perror.FATAL,
			perror.ProtocolViolation,
			"Expected password",
		)
	}

	pw, ok := reader.String()
	if !ok {
		return ErrBadPacketFormat
	}

	if !md5.Check(username, password, salt, pw) {
		return perror.New(
			perror.FATAL,
			perror.InvalidPassword,
			"Invalid password",
		)
	}

	return nil
}

func (T *Client) authenticationCleartext(password string) perror.Error {
	var builder packet.Builder
	builder.Type(packet.Authentication)
	builder.Uint32(3)

	err := T.Write(builder.Raw())
	if err != nil {
		return WrapError(err)
	}

	// read password
	pkt, err := T.Read()
	if err != nil {
		return WrapError(err)
	}

	reader := packet.MakeReader(pkt)
	if reader.Type() != packet.AuthenticationResponse {
		return perror.New(
			perror.FATAL,
			perror.ProtocolViolation,
			"Expected password",
		)
	}

	pw, ok := reader.String()
	if !ok {
		return ErrBadPacketFormat
	}

	if pw != password {
		return perror.New(
			perror.FATAL,
			perror.InvalidPassword,
			"Invalid password",
		)
	}

	return nil
}

func (T *Client) accept() perror.Error {
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
	perr := T.authenticationSASL("test", "password")
	if perr != nil {
		return perr
	}

	// send auth ok
	builder := packet.Builder{}
	builder.Type(packet.Authentication)
	builder.Uint32(0)

	err := T.Write(builder.Raw())
	if err != nil {
		return WrapError(err)
	}

	// send backend key data
	_, err = rand.Read(T.cancellationKey[:])
	if err != nil {
		return WrapError(err)
	}
	builder = packet.Builder{}
	builder.Type(packet.BackendKeyData)
	builder.Bytes(T.cancellationKey[:])

	err = T.Write(builder.Raw())
	if err != nil {
		return WrapError(err)
	}

	// send ready for query
	builder = packet.Builder{}
	builder.Type(packet.ReadyForQuery)
	builder.Uint8('I')

	err = T.Write(builder.Raw())
	if err != nil {
		return WrapError(err)
	}

	return nil
}

func (T *Client) Close(err perror.Error) {
	if err != nil {
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

		_ = T.Write(builder.Raw())
	}
	_ = T.conn.Close()
}

var _ frontend.Client = (*Client)(nil)
