package backends

import (
	"errors"
	"io"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func authenticationSASLChallenge(ctx *AcceptContext, encoder auth.SASLEncoder) (done bool, err error) {
	ctx.Packet, err = ctx.Conn.ReadPacket(true, ctx.Packet)
	if err != nil {
		return
	}

	if ctx.Packet.Type() != packets.TypeAuthentication {
		err = ErrUnexpectedPacket
		return
	}

	var method int32
	p := ctx.Packet.ReadInt32(&method)

	switch method {
	case 11:
		// challenge
		var response []byte
		response, err = encoder.Write(p)
		if err != nil {
			return
		}

		resp := packets.AuthenticationResponse(response)
		ctx.Packet = resp.IntoPacket(ctx.Packet)
		err = ctx.Conn.WritePacket(ctx.Packet)
		return
	case 12:
		// finish
		_, err = encoder.Write(p)
		if err != io.EOF {
			if err == nil {
				err = errors.New("expected EOF")
			}
			return
		}

		return true, nil
	default:
		err = ErrUnexpectedPacket
		return
	}
}

func authenticationSASL(ctx *AcceptContext, mechanisms []string, creds auth.SASLClient) error {
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
	ctx.Packet = saslInitialResponse.IntoPacket(ctx.Packet)
	err = ctx.Conn.WritePacket(ctx.Packet)
	if err != nil {
		return err
	}

	// challenge loop
	for {
		var done bool
		done, err = authenticationSASLChallenge(ctx, encoder)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	return nil
}

func authenticationMD5(ctx *AcceptContext, salt [4]byte, creds auth.MD5Client) error {
	pw := packets.PasswordMessage{
		Password: creds.EncodeMD5(salt),
	}
	ctx.Packet = pw.IntoPacket(ctx.Packet)
	err := ctx.Conn.WritePacket(ctx.Packet)
	if err != nil {
		return err
	}
	return nil
}

func authenticationCleartext(ctx *AcceptContext, creds auth.CleartextClient) error {
	pw := packets.PasswordMessage{
		Password: creds.EncodeCleartext(),
	}
	ctx.Packet = pw.IntoPacket(ctx.Packet)
	err := ctx.Conn.WritePacket(ctx.Packet)
	if err != nil {
		return err
	}
	return nil
}

func authentication(ctx *AcceptContext) (done bool, err error) {
	var method int32
	ctx.Packet.ReadInt32(&method)
	// they have more authentication methods than there are pokemon
	switch method {
	case 0:
		// we're good to go, that was easy
		return true, nil
	case 2:
		err = errors.New("kerberos v5 is not supported")
		return
	case 3:
		c, ok := ctx.Options.Credentials.(auth.CleartextClient)
		if !ok {
			return false, auth.ErrMethodNotSupported
		}
		return false, authenticationCleartext(ctx, c)
	case 5:
		var md5 packets.AuthenticationMD5
		if !md5.ReadFromPacket(ctx.Packet) {
			err = ErrBadFormat
			return
		}

		c, ok := ctx.Options.Credentials.(auth.MD5Client)
		if !ok {
			return false, auth.ErrMethodNotSupported
		}
		return false, authenticationMD5(ctx, md5.Salt, c)
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
		if !sasl.ReadFromPacket(ctx.Packet) {
			err = ErrBadFormat
			return
		}

		c, ok := ctx.Options.Credentials.(auth.SASLClient)
		if !ok {
			return false, auth.ErrMethodNotSupported
		}
		return false, authenticationSASL(ctx, sasl.Mechanisms, c)
	default:
		err = errors.New("unknown authentication method")
		return
	}
}

func startup0(ctx *AcceptContext) (done bool, err error) {
	ctx.Packet, err = ctx.Conn.ReadPacket(true, ctx.Packet)
	if err != nil {
		return
	}

	switch ctx.Packet.Type() {
	case packets.TypeErrorResponse:
		var err2 packets.ErrorResponse
		if !err2.ReadFromPacket(ctx.Packet) {
			err = ErrBadFormat
		} else {
			err = errors.New(err2.Error.String())
		}
		return
	case packets.TypeAuthentication:
		return authentication(ctx)
	case packets.TypeNegotiateProtocolVersion:
		// we only support protocol 3.0 for now
		err = errors.New("server wanted to negotiate protocol version")
		return
	default:
		err = ErrUnexpectedPacket
		return
	}
}

func startup1(ctx *AcceptContext, params *AcceptParams) (done bool, err error) {
	ctx.Packet, err = ctx.Conn.ReadPacket(true, ctx.Packet)
	if err != nil {
		return
	}

	switch ctx.Packet.Type() {
	case packets.TypeBackendKeyData:
		ctx.Packet.ReadBytes(params.BackendKey[:])
		return false, nil
	case packets.TypeParameterStatus:
		var ps packets.ParameterStatus
		if !ps.ReadFromPacket(ctx.Packet) {
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
		if !err2.ReadFromPacket(ctx.Packet) {
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

func enableSSL(ctx *AcceptContext) (bool, error) {
	ctx.Packet = ctx.Packet.Reset(0, 4)
	ctx.Packet = ctx.Packet.AppendUint16(1234)
	ctx.Packet = ctx.Packet.AppendUint16(5679)
	if err := ctx.Conn.WritePacket(ctx.Packet); err != nil {
		return false, err
	}

	byteReader, ok := ctx.Conn.(io.ByteReader)
	if !ok {
		return false, errors.New("server must be io.ByteReader to enable ssl")
	}

	// read byte to see if ssl is allowed
	yn, err := byteReader.ReadByte()
	if err != nil {
		return false, err
	}

	if yn != 'S' {
		// not supported
		return false, nil
	}

	sslClient, ok := ctx.Conn.(fed.SSLClient)
	if !ok {
		return false, errors.New("server must be fed.SSLClient to enable ssl")
	}

	if err = sslClient.EnableSSLClient(ctx.Options.SSLConfig); err != nil {
		return false, err
	}

	return true, nil
}

func Accept(ctx *AcceptContext) (AcceptParams, error) {
	username := ctx.Options.Username

	if ctx.Options.Database == "" {
		ctx.Options.Database = username
	}

	var params AcceptParams

	if ctx.Options.SSLMode.ShouldAttempt() {
		var err error
		params.SSLEnabled, err = enableSSL(ctx)
		if err != nil {
			return AcceptParams{}, err
		}
		if !params.SSLEnabled && ctx.Options.SSLMode.IsRequired() {
			return AcceptParams{}, errors.New("server rejected SSL encryption")
		}
	}

	size := 4 + len("user") + 1 + len(username) + 1 + len("database") + 1 + len(ctx.Options.Database) + 1
	for key, value := range ctx.Options.StartupParameters {
		size += len(key.String()) + len(value) + 2
	}
	size += 1

	ctx.Packet = ctx.Packet.Reset(0, size)
	ctx.Packet = ctx.Packet.AppendUint16(3)
	ctx.Packet = ctx.Packet.AppendUint16(0)
	ctx.Packet = ctx.Packet.AppendString("user")
	ctx.Packet = ctx.Packet.AppendString(username)
	ctx.Packet = ctx.Packet.AppendString("database")
	ctx.Packet = ctx.Packet.AppendString(ctx.Options.Database)
	for key, value := range ctx.Options.StartupParameters {
		ctx.Packet = ctx.Packet.AppendString(key.String())
		ctx.Packet = ctx.Packet.AppendString(value)
	}
	ctx.Packet = ctx.Packet.AppendUint8(0)

	err := ctx.Conn.WritePacket(ctx.Packet)
	if err != nil {
		return AcceptParams{}, err
	}

	for {
		var done bool
		done, err = startup0(ctx)
		if err != nil {
			return AcceptParams{}, err
		}
		if done {
			break
		}
	}

	for {
		var done bool
		done, err = startup1(ctx, &params)
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
