package frontends

import (
	"crypto/rand"
	"net"
	"strings"

	"pggat2/lib/auth/md5"
	"pggat2/lib/auth/sasl"
	"pggat2/lib/eqp"
	"pggat2/lib/frontend"
	"pggat2/lib/perror"
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	packets "pggat2/lib/pnet/packet/packets/v3.0"
	"pggat2/lib/util/decorator"
)

type Client struct {
	noCopy decorator.NoCopy

	conn net.Conn

	reader pnet.IOReader
	writer pnet.IOWriter

	user     string
	database string

	// cancellation key data
	cancellationKey    [8]byte
	parameters         map[string]string
	preparedStatements map[string]eqp.PreparedStatement
	portals            map[string]eqp.Portal
}

func NewClient(conn net.Conn) *Client {
	client := &Client{
		conn:               conn,
		reader:             pnet.MakeIOReader(conn),
		writer:             pnet.MakeIOWriter(conn),
		parameters:         make(map[string]string),
		preparedStatements: make(map[string]eqp.PreparedStatement),
		portals:            make(map[string]eqp.Portal),
	}
	err := client.accept()
	if err != nil {
		client.Close(err)
	}
	return client
}

func (T *Client) GetPreparedStatement(name string) (eqp.PreparedStatement, bool) {
	v, ok := T.preparedStatements[name]
	return v, ok
}

func (T *Client) GetPortal(name string) (eqp.Portal, bool) {
	v, ok := T.portals[name]
	return v, ok
}

func (T *Client) startup0() (bool, perror.Error) {
	pkt, err := T.ReadUntyped()
	if err != nil {
		return false, perror.Wrap(err)
	}

	majorVersion, ok := pkt.Uint16()
	if !ok {
		return false, pnet.ErrBadPacketFormat
	}
	minorVersion, ok := pkt.Uint16()
	if !ok {
		return false, pnet.ErrBadPacketFormat
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
			err = T.writer.WriteByte('N')
			return false, perror.Wrap(err)
		case 5680:
			// GSSAPI is not supported yet
			err = T.writer.WriteByte('N')
			return false, perror.Wrap(err)
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
			return false, pnet.ErrBadPacketFormat
		}
		if key == "" {
			break
		}

		value, ok := pkt.String()
		if !ok {
			return false, pnet.ErrBadPacketFormat
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
		packets.WriteNegotiateProtocolVersion(out, 0, unsupportedOptions)

		err = out.Send()
		if err != nil {
			return false, perror.Wrap(err)
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

func (T *Client) authenticationSASLInitial(username, password string) (sasl.Server, []byte, bool, perror.Error) {
	// check which authentication method the client wants
	in, err := T.Read()
	if err != nil {
		return nil, nil, false, perror.Wrap(err)
	}
	mechanism, initialResponse, ok := packets.ReadSASLInitialResponse(in)
	if !ok {
		return nil, nil, false, pnet.ErrBadPacketFormat
	}

	tool, err := sasl.NewServer(mechanism, username, password)
	if err != nil {
		return nil, nil, false, perror.Wrap(err)
	}

	resp, done, err := tool.InitialResponse(initialResponse)
	if err != nil {
		return nil, nil, false, perror.Wrap(err)
	}
	return tool, resp, done, nil
}

func (T *Client) authenticationSASLContinue(tool sasl.Server) ([]byte, bool, perror.Error) {
	in, err := T.Read()
	if err != nil {
		return nil, false, perror.Wrap(err)
	}
	clientResp, ok := packets.ReadAuthenticationResponse(in)
	if !ok {
		return nil, false, pnet.ErrProtocolError
	}

	resp, done, err := tool.Continue(clientResp)
	if err != nil {
		return nil, false, perror.Wrap(err)
	}
	return resp, done, nil
}

func (T *Client) authenticationSASL(username, password string) perror.Error {
	out := T.Write()
	packets.WriteAuthenticationSASL(out, sasl.Mechanisms)
	err := out.Send()
	if err != nil {
		return perror.Wrap(err)
	}

	tool, resp, done, perr := T.authenticationSASLInitial(username, password)

	for {
		if perr != nil {
			return perr
		}
		if done {
			out = T.Write()
			packets.WriteAuthenticationSASLFinal(out, resp)
			err = out.Send()
			if err != nil {
				return perror.Wrap(err)
			}
			break
		} else {
			out = T.Write()
			packets.WriteAuthenticationSASLContinue(out, resp)
			err = out.Send()
			if err != nil {
				return perror.Wrap(err)
			}
		}

		resp, done, perr = T.authenticationSASLContinue(tool)
	}

	return nil
}

func (T *Client) authenticationMD5(username, password string) perror.Error {
	var salt [4]byte
	_, err := rand.Read(salt[:])
	if err != nil {
		return perror.Wrap(err)
	}

	// password time
	out := T.Write()
	packets.WriteAuthenticationMD5(out, salt)

	err = out.Send()
	if err != nil {
		return perror.Wrap(err)
	}

	// read password
	in, err := T.Read()
	if err != nil {
		return perror.Wrap(err)
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
		return pnet.ErrBadPacketFormat
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
	out := T.Write()
	packets.WriteAuthenticationCleartext(out)

	err := out.Send()
	if err != nil {
		return perror.Wrap(err)
	}

	// read password
	in, err := T.Read()
	if err != nil {
		return perror.Wrap(err)
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
		return pnet.ErrBadPacketFormat
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

func (T *Client) updateParameter(name, value string) perror.Error {
	out := T.Write()
	packets.WriteParameterStatus(out, name, value)
	return perror.Wrap(out.Send())
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
	packets.WriteAuthenticationOk(out)

	err := out.Send()
	if err != nil {
		return perror.Wrap(err)
	}

	perr = T.updateParameter("DateStyle", "ISO, MDY")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("IntervalStyle", "postgres")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("TimeZone", "America/Chicago")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("application_name", "")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("client_encoding", "UTF8")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("default_transaction_read_only", "off")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("in_hot_standby", "off")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("integer_datetimes", "on")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("is_superuser", "on")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("server_encoding", "UTF8")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("server_version", "14.5")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("session_authorization", "postgres")
	if perr != nil {
		return perr
	}
	perr = T.updateParameter("standard_conforming_strings", "on")
	if perr != nil {
		return perr
	}

	// send backend key data
	_, err = rand.Read(T.cancellationKey[:])
	if err != nil {
		return perror.Wrap(err)
	}
	out = T.Write()
	packets.WriteBackendKeyData(out, T.cancellationKey)

	err = out.Send()
	if err != nil {
		return perror.Wrap(err)
	}

	// send ready for query
	out = T.Write()
	packets.WriteReadyForQuery(out, 'I')

	err = out.Send()
	if err != nil {
		return perror.Wrap(err)
	}

	return nil
}

func (T *Client) Wait() error {
	_, err := T.conn.Read(nil)
	return err
}

func (T *Client) Write() packet.Out {
	return T.writer.Write()
}

func (T *Client) Send(typ packet.Type, payload []byte) error {
	return T.writer.Send(typ, payload)
}

func (T *Client) Read() (packet.In, error) {
	return T.reader.Read()
}

func (T *Client) ReadUntyped() (packet.In, error) {
	return T.reader.ReadUntyped()
}

func (T *Client) Close(err perror.Error) {
	if err != nil {
		out := T.Write()
		packets.WriteErrorResponse(out, err)
		_ = out.Send()
	}
	_ = T.conn.Close()
}

var _ frontend.Client = (*Client)(nil)
