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
	"pggat2/lib/util/decorator"
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
	noCopy decorator.NoCopy

	conn net.Conn

	pnet.IOReader
	pnet.IOWriter

	user     string
	database string

	// cancellation key data
	cancellationKey [8]byte
	parameters      map[string]string
}

func NewClient(conn net.Conn) *Client {
	client := &Client{
		conn:       conn,
		IOReader:   pnet.MakeIOReader(conn),
		IOWriter:   pnet.MakeIOWriter(conn),
		parameters: make(map[string]string),
	}
	err := client.accept()
	if err != nil {
		client.Close(err)
		return nil
	}
	return client
}

func negotiateProtocolVersionPacket(pkt packet.Out, unsupportedOptions []string) {
	pkt.Type(packet.NegotiateProtocolVersion)
	pkt.Int32(0)
	pkt.Int32(int32(len(unsupportedOptions)))
	for _, v := range unsupportedOptions {
		pkt.String(v)
	}
}

func (T *Client) startup0() (bool, perror.Error) {
	pkt, err := T.ReadUntyped()
	if err != nil {
		return false, perror.WrapError(err)
	}

	majorVersion, ok := pkt.Uint16()
	if !ok {
		return false, ErrBadPacketFormat
	}
	minorVersion, ok := pkt.Uint16()
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
			return false, perror.WrapError(err)
		case 5680:
			// GSSAPI is not supported yet
			err = T.WriteByte('N')
			return false, perror.WrapError(err)
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
		key, ok := pkt.String()
		if !ok {
			return false, ErrBadPacketFormat
		}
		if key == "" {
			break
		}

		value, ok := pkt.String()
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
		out := T.Write()
		negotiateProtocolVersionPacket(out, unsupportedOptions)

		err = out.Send()
		if err != nil {
			return false, perror.WrapError(err)
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

func authenticationSASLPacket(pkt packet.Out) {
	pkt.Type(packet.Authentication)
	pkt.Int32(10)
	for _, mechanism := range sasl.Mechanisms {
		pkt.String(mechanism)
	}
	pkt.String("")
}

func authenticationSASLContinuePacket(pkt packet.Out, resp []byte) {
	pkt.Type(packet.Authentication)
	pkt.Int32(11)
	pkt.Bytes(resp)
}

func authenticationSASLFinalPacket(pkt packet.Out, resp []byte) {
	pkt.Type(packet.Authentication)
	pkt.Int32(12)
	pkt.Bytes(resp)
}

func (T *Client) authenticationSASLInitial(username, password string) (sasl.Server, []byte, bool, perror.Error) {
	// check which authentication method the client wants
	in, err := T.Read()
	if err != nil {
		return nil, nil, false, perror.WrapError(err)
	}
	if in.Type() != packet.AuthenticationResponse {
		return nil, nil, false, ErrBadPacketFormat
	}

	mechanism, ok := in.String()
	if !ok {
		return nil, nil, false, ErrBadPacketFormat
	}
	tool, err := sasl.NewServer(mechanism, username, password)
	if err != nil {
		return nil, nil, false, perror.WrapError(err)
	}
	_, ok = in.Int32()
	if !ok {
		return nil, nil, false, ErrBadPacketFormat
	}

	resp, done, err := tool.InitialResponse(in.Remaining())
	if err != nil {
		return nil, nil, false, perror.WrapError(err)
	}
	return tool, resp, done, nil
}

func (T *Client) authenticationSASLContinue(tool sasl.Server) ([]byte, bool, perror.Error) {
	in, err := T.Read()
	if err != nil {
		return nil, false, perror.WrapError(err)
	}
	if in.Type() != packet.AuthenticationResponse {
		return nil, false, ErrProtocolError
	}

	resp, done, err := tool.Continue(in.Full())
	if err != nil {
		return nil, false, perror.WrapError(err)
	}
	return resp, done, nil
}

func (T *Client) authenticationSASL(username, password string) perror.Error {
	out := T.Write()
	authenticationSASLPacket(out)
	err := out.Send()
	if err != nil {
		return perror.WrapError(err)
	}

	tool, resp, done, perr := T.authenticationSASLInitial(username, password)

	for {
		if perr != nil {
			return perr
		}
		if done {
			out = T.Write()
			authenticationSASLFinalPacket(out, resp)
			err = out.Send()
			if err != nil {
				return perror.WrapError(err)
			}
			break
		} else {
			out = T.Write()
			authenticationSASLContinuePacket(out, resp)
			err = out.Send()
			if err != nil {
				return perror.WrapError(err)
			}
		}

		resp, done, perr = T.authenticationSASLContinue(tool)
	}

	return nil
}

func authenticationMD5Packet(pkt packet.Out, salt [4]byte) {
	pkt.Type(packet.Authentication)
	pkt.Uint32(5)
	pkt.Bytes(salt[:])
}

func (T *Client) authenticationMD5(username, password string) perror.Error {
	var salt [4]byte
	_, err := rand.Read(salt[:])
	if err != nil {
		return perror.WrapError(err)
	}

	// password time
	out := T.Write()
	authenticationMD5Packet(out, salt)

	err = out.Send()
	if err != nil {
		return perror.WrapError(err)
	}

	// read password
	in, err := T.Read()
	if err != nil {
		return perror.WrapError(err)
	}

	if in.Type() != packet.AuthenticationResponse {
		return perror.New(
			perror.FATAL,
			perror.ProtocolViolation,
			"Expected password",
		)
	}

	pw, ok := in.String()
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

func authenticationCleartextPacket(pkt packet.Out) {
	pkt.Type(packet.Authentication)
	pkt.Uint32(3)
}

func (T *Client) authenticationCleartext(password string) perror.Error {
	out := T.Write()
	authenticationCleartextPacket(out)

	err := out.Send()
	if err != nil {
		return perror.WrapError(err)
	}

	// read password
	in, err := T.Read()
	if err != nil {
		return perror.WrapError(err)
	}

	if in.Type() != packet.AuthenticationResponse {
		return perror.New(
			perror.FATAL,
			perror.ProtocolViolation,
			"Expected password",
		)
	}

	pw, ok := in.String()
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

func authenticationOkPacket(pkt packet.Out) {
	pkt.Type(packet.Authentication)
	pkt.Uint32(0)
}

func backendKeyDataPacket(pkt packet.Out, cancellationKey [8]byte) {
	pkt.Type(packet.BackendKeyData)
	pkt.Bytes(cancellationKey[:])
}

func readyForQueryPacket(pkt packet.Out, state byte) {
	pkt.Type(packet.ReadyForQuery)
	pkt.Uint8(state)
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
	out := T.Write()
	authenticationOkPacket(out)

	err := out.Send()
	if err != nil {
		return perror.WrapError(err)
	}

	// send backend key data
	_, err = rand.Read(T.cancellationKey[:])
	if err != nil {
		return perror.WrapError(err)
	}
	out = T.Write()
	backendKeyDataPacket(out, T.cancellationKey)

	err = out.Send()
	if err != nil {
		return perror.WrapError(err)
	}

	// send ready for query
	out = T.Write()
	readyForQueryPacket(out, 'I')

	err = out.Send()
	if err != nil {
		return perror.WrapError(err)
	}

	return nil
}

func (T *Client) Close(err perror.Error) {
	if err != nil {
		out := T.Write()
		out.Type(packet.ErrorResponse)

		out.Uint8('S')
		out.String(string(err.Severity()))

		out.Uint8('C')
		out.String(string(err.Code()))

		out.Uint8('M')
		out.String(err.Message())

		for _, field := range err.Extra() {
			out.Uint8(uint8(field.Type))
			out.String(field.Value)
		}

		out.Uint8(0)

		_ = out.Send()
	}
	_ = T.conn.Close()
}

var _ frontend.Client = (*Client)(nil)
