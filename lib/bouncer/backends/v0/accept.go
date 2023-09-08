package backends

import (
	"crypto/tls"
	"errors"

	"pggat/lib/auth"
	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/util/strutil"
)

func authenticationSASLChallenge(server fed.Conn, encoder auth.SASLEncoder) (done bool, err error) {
	var packet fed.Packet
	packet, err = server.ReadPacket(true)
	if err != nil {
		return
	}

	if packet.Type() != packets.TypeAuthentication {
		err = ErrUnexpectedPacket
		return
	}

	var method int32
	p := packet.ReadInt32(&method)

	switch method {
	case 11:
		// challenge
		var response []byte
		response, err = encoder.Write(p)
		if err != nil {
			return
		}

		resp := packets.AuthenticationResponse(response)
		err = server.WritePacket(resp.IntoPacket())
		return
	case 12:
		// finish
		_, err = encoder.Write(p)
		if err != nil {
			return
		}

		return true, nil
	default:
		err = ErrUnexpectedPacket
		return
	}
}

func authenticationSASL(server fed.Conn, mechanisms []string, creds auth.SASL) error {
	mechanism, encoder, err := creds.EncodeSASL(mechanisms)
	if err != nil {
		return err
	}
	initialResponse, err := encoder.Write(nil)
	if err != nil {
		return err
	}

	saslInitialResponse := packets.SASLInitialResponse{
		Mechanism:       mechanism,
		InitialResponse: initialResponse,
	}
	err = server.WritePacket(saslInitialResponse.IntoPacket())
	if err != nil {
		return err
	}

	// challenge loop
	for {
		var done bool
		done, err = authenticationSASLChallenge(server, encoder)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	return nil
}

func authenticationMD5(server fed.Conn, salt [4]byte, creds auth.MD5) error {
	pw := packets.PasswordMessage{
		Password: creds.EncodeMD5(salt),
	}
	err := server.WritePacket(pw.IntoPacket())
	if err != nil {
		return err
	}
	return nil
}

func authenticationCleartext(server fed.Conn, creds auth.Cleartext) error {
	pw := packets.PasswordMessage{
		Password: creds.EncodeCleartext(),
	}
	err := server.WritePacket(pw.IntoPacket())
	if err != nil {
		return err
	}
	return nil
}

func authentication(server fed.Conn, creds auth.Credentials, packet fed.Packet) (done bool, err error) {
	var method int32
	packet.ReadInt32(&method)
	// they have more authentication methods than there are pokemon
	switch method {
	case 0:
		// we're good to go, that was easy
		return true, nil
	case 2:
		err = errors.New("kerberos v5 is not supported")
		return
	case 3:
		c, ok := creds.(auth.Cleartext)
		if !ok {
			return false, auth.ErrMethodNotSupported
		}
		return false, authenticationCleartext(server, c)
	case 5:
		var md5 packets.AuthenticationMD5
		if !md5.ReadFromPacket(packet) {
			err = ErrBadFormat
			return
		}

		c, ok := creds.(auth.MD5)
		if !ok {
			return false, auth.ErrMethodNotSupported
		}
		return false, authenticationMD5(server, md5.Salt, c)
	case 6:
		err = errors.New("scm credential is not supported")
		return
	case 7:
		err = errors.New("gss is not supported")
		return
	case 9:
		err = errors.New("sspi is not supported")
		return
	case 10:
		// read list of mechanisms
		var sasl packets.AuthenticationSASL
		if !sasl.ReadFromPacket(packet) {
			err = ErrBadFormat
			return
		}

		c, ok := creds.(auth.SASL)
		if !ok {
			return false, auth.ErrMethodNotSupported
		}
		return false, authenticationSASL(server, sasl.Mechanisms, c)
	default:
		err = errors.New("unknown authentication method")
		return
	}
}

func startup0(server fed.Conn, creds auth.Credentials) (done bool, err error) {
	var packet fed.Packet
	packet, err = server.ReadPacket(true)
	if err != nil {
		return
	}

	switch packet.Type() {
	case packets.TypeErrorResponse:
		var err2 packets.ErrorResponse
		if !err2.ReadFromPacket(packet) {
			err = ErrBadFormat
		} else {
			err = errors.New(err2.Error.String())
		}
		return
	case packets.TypeAuthentication:
		return authentication(server, creds, packet)
	case packets.TypeNegotiateProtocolVersion:
		// we only support protocol 3.0 for now
		err = errors.New("server wanted to negotiate protocol version")
		return
	default:
		err = ErrUnexpectedPacket
		return
	}
}

func startup1(conn fed.Conn, params *AcceptParams) (done bool, err error) {
	var packet fed.Packet
	packet, err = conn.ReadPacket(true)
	if err != nil {
		return
	}

	switch packet.Type() {
	case packets.TypeBackendKeyData:
		packet.ReadBytes(params.BackendKey[:])
		return false, nil
	case packets.TypeParameterStatus:
		var ps packets.ParameterStatus
		if !ps.ReadFromPacket(packet) {
			err = ErrBadFormat
			return
		}
		ikey := strutil.MakeCIString(ps.Key)
		if params.InitialParameters == nil {
			params.InitialParameters = make(map[strutil.CIString]string)
		}
		params.InitialParameters[ikey] = ps.Value
		return false, nil
	case packets.TypeReadyForQuery:
		return true, nil
	case packets.TypeErrorResponse:
		var err2 packets.ErrorResponse
		if !err2.ReadFromPacket(packet) {
			err = ErrBadFormat
		} else {
			err = errors.New(err2.Error.String())
		}
		return
	case packets.TypeNoticeResponse:
		// TODO(garet) do something with notice
		return false, nil
	default:
		err = ErrUnexpectedPacket
		return
	}
}

func enableSSL(server fed.Conn, config *tls.Config) (bool, error) {
	packet := fed.NewPacket(0, 4)
	packet = packet.AppendUint16(1234)
	packet = packet.AppendUint16(5679)
	if err := server.WritePacket(packet); err != nil {
		return false, err
	}

	// read byte to see if ssl is allowed
	yn, err := server.ReadByte()
	if err != nil {
		return false, err
	}

	if yn != 'S' {
		// not supported
		return false, nil
	}

	if err = server.EnableSSLClient(config); err != nil {
		return false, err
	}

	return true, nil
}

func Accept(server fed.Conn, options AcceptOptions) (AcceptParams, error) {
	username := options.Credentials.GetUsername()

	if options.Database == "" {
		options.Database = username
	}

	var params AcceptParams

	if options.SSLMode.ShouldAttempt() {
		var err error
		params.SSLEnabled, err = enableSSL(server, options.SSLConfig)
		if err != nil {
			return AcceptParams{}, err
		}
		if !params.SSLEnabled && options.SSLMode.IsRequired() {
			return AcceptParams{}, errors.New("server rejected SSL encryption")
		}
	}

	size := 4 + len("user") + 1 + len(username) + 1 + len("database") + 1 + len(options.Database) + 1
	for key, value := range options.StartupParameters {
		size += len(key.String()) + len(value) + 2
	}
	size += 1

	packet := fed.NewPacket(0, size)
	packet = packet.AppendUint16(3)
	packet = packet.AppendUint16(0)
	packet = packet.AppendString("user")
	packet = packet.AppendString(username)
	packet = packet.AppendString("database")
	packet = packet.AppendString(options.Database)
	for key, value := range options.StartupParameters {
		packet = packet.AppendString(key.String())
		packet = packet.AppendString(value)
	}
	packet = packet.AppendUint8(0)

	err := server.WritePacket(packet)
	if err != nil {
		return AcceptParams{}, err
	}

	for {
		var done bool
		done, err = startup0(server, options.Credentials)
		if err != nil {
			return AcceptParams{}, err
		}
		if done {
			break
		}
	}

	for {
		var done bool
		done, err = startup1(server, &params)
		if err != nil {
			return AcceptParams{}, err
		}
		if done {
			break
		}
	}

	// startup complete, connection is ready for queries
	return params, nil
}
