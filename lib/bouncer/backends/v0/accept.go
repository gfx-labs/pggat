package backends

import (
	"crypto/tls"
	"errors"
	"io"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func authenticationSASLChallenge(ctx *acceptContext, encoder auth.SASLEncoder) (done bool, err error) {
	var packet fed.Packet
	packet, err = ctx.Conn.ReadPacket(true)
	if err != nil {
		return
	}

	if packet.Type() != packets.TypeAuthentication {
		err = ErrUnexpectedPacket(packet.Type())
		return
	}

	var p packets.Authentication
	err = fed.ToConcrete(&p, packet)
	if err != nil {
		return
	}

	switch mode := p.Mode.(type) {
	case *packets.AuthenticationPayloadSASLContinue:
		// challenge
		var response []byte
		response, err = encoder.Write(*mode)
		if err != nil {
			return
		}

		resp := packets.SASLResponse(response)
		err = ctx.Conn.WritePacket(&resp)
		return
	case *packets.AuthenticationPayloadSASLFinal:
		// finish
		_, err = encoder.Write(*mode)
		if err != io.EOF {
			if err == nil {
				err = errors.New("expected EOF")
			}
			return
		}

		return true, nil
	default:
		err = ErrUnexpectedAuthenticationResponse
		return
	}
}

func authenticationSASL(ctx *acceptContext, mechanisms []string, creds auth.SASLClient) error {
	mechanism, encoder, err := creds.EncodeSASL(mechanisms)
	if err != nil {
		return err
	}
	initialResponse, err := encoder.Write(nil)
	if err != nil {
		return err
	}

	saslInitialResponse := packets.SASLInitialResponse{
		Mechanism:             mechanism,
		InitialClientResponse: initialResponse,
	}
	err = ctx.Conn.WritePacket(&saslInitialResponse)
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

func authenticationMD5(ctx *acceptContext, salt [4]byte, creds auth.MD5Client) error {
	pw := packets.PasswordMessage(creds.EncodeMD5(salt))
	err := ctx.Conn.WritePacket(&pw)
	if err != nil {
		return err
	}
	return nil
}

func authenticationCleartext(ctx *acceptContext, creds auth.CleartextClient) error {
	pw := packets.PasswordMessage(creds.EncodeCleartext())
	err := ctx.Conn.WritePacket(&pw)
	if err != nil {
		return err
	}
	return nil
}

func authentication(ctx *acceptContext, p *packets.Authentication) (done bool, err error) {
	// they have more authentication methods than there are pokemon
	switch mode := p.Mode.(type) {
	case *packets.AuthenticationPayloadOk:
		// we're good to go, that was easy
		ctx.Conn.Authenticated = true
		return true, nil
	case *packets.AuthenticationPayloadKerberosV5:
		err = errors.New("kerberos v5 is not supported")
		return
	case *packets.AuthenticationPayloadCleartextPassword:
		c, ok := ctx.Options.Credentials.(auth.CleartextClient)
		if !ok {
			return false, auth.ErrMethodNotSupported
		}
		return false, authenticationCleartext(ctx, c)
	case *packets.AuthenticationPayloadMD5Password:
		c, ok := ctx.Options.Credentials.(auth.MD5Client)
		if !ok {
			return false, auth.ErrMethodNotSupported
		}
		return false, authenticationMD5(ctx, *mode, c)
	case *packets.AuthenticationPayloadGSS:
		err = errors.New("gss is not supported")
		return
	case *packets.AuthenticationPayloadSSPI:
		err = errors.New("sspi is not supported")
		return
	case *packets.AuthenticationPayloadSASL:
		c, ok := ctx.Options.Credentials.(auth.SASLClient)
		if !ok {
			return false, auth.ErrMethodNotSupported
		}

		var mechanisms = make([]string, 0, len(*mode))
		for _, m := range *mode {
			mechanisms = append(mechanisms, m.Method)
		}

		return false, authenticationSASL(ctx, mechanisms, c)
	default:
		err = errors.New("unknown authentication method")
		return
	}
}

func startup0(ctx *acceptContext) (done bool, err error) {
	var packet fed.Packet
	packet, err = ctx.Conn.ReadPacket(true)
	if err != nil {
		return
	}

	switch packet.Type() {
	case packets.TypeErrorResponse:
		var p packets.ErrorResponse
		err = fed.ToConcrete(&p, packet)
		if err != nil {
			return
		}
		err = perror.FromPacket(&p)
		return
	case packets.TypeAuthentication:
		var p packets.Authentication
		err = fed.ToConcrete(&p, packet)
		if err != nil {
			return
		}
		return authentication(ctx, &p)
	case packets.TypeNegotiateProtocolVersion:
		// we only support protocol 3.0 for now
		err = errors.New("server wanted to negotiate protocol version")
		return
	default:
		err = ErrUnexpectedPacket(packet.Type())
		return
	}
}

func startup1(ctx *acceptContext) (done bool, err error) {
	var packet fed.Packet
	packet, err = ctx.Conn.ReadPacket(true)
	if err != nil {
		return
	}

	switch packet.Type() {
	case packets.TypeBackendKeyData:
		var p packets.BackendKeyData
		err = fed.ToConcrete(&p, packet)
		if err != nil {
			return
		}
		ctx.Conn.BackendKey.SecretKey = p.SecretKey
		ctx.Conn.BackendKey.ProcessID = p.ProcessID

		return false, nil
	case packets.TypeParameterStatus:
		var p packets.ParameterStatus
		err = fed.ToConcrete(&p, packet)
		if err != nil {
			return
		}
		ikey := strutil.MakeCIString(p.Key)
		if ctx.Conn.InitialParameters == nil {
			ctx.Conn.InitialParameters = make(map[strutil.CIString]string)
		}
		ctx.Conn.InitialParameters[ikey] = p.Value
		return false, nil
	case packets.TypeReadyForQuery:
		return true, nil
	case packets.TypeErrorResponse:
		var p packets.ErrorResponse
		err = fed.ToConcrete(&p, packet)
		if err != nil {
			return
		}
		err = perror.FromPacket(&p)
		return
	case packets.TypeNoticeResponse:
		// TODO(garet) do something with notice
		return false, nil
	default:
		err = ErrUnexpectedPacket(packet.Type())
		return
	}
}

func enableSSL(ctx *acceptContext) (bool, error) {
	p := packets.Startup{
		Mode: &packets.StartupPayloadControl{
			Mode: &packets.StartupPayloadControlPayloadSSL{},
		},
	}

	if err := ctx.Conn.WritePacket(&p); err != nil {
		return false, err
	}

	// read byte to see if ssl is allowed
	yn, err := ctx.Conn.ReadByte()
	if err != nil {
		return false, err
	}

	if yn != 'S' {
		// not supported
		return false, nil
	}

	if err = ctx.Conn.EnableSSL(ctx.Options.SSLConfig, true); err != nil {
		return false, err
	}

	return true, nil
}

func accept(ctx *acceptContext) error {
	username := ctx.Options.Username

	if ctx.Options.Database == "" {
		ctx.Options.Database = username
	}

	if ctx.Options.SSLMode.ShouldAttempt() {
		sslEnabled, err := enableSSL(ctx)
		if err != nil {
			return err
		}
		if !sslEnabled && ctx.Options.SSLMode.IsRequired() {
			return errors.New("server rejected SSL encryption")
		}
	}

	m := packets.StartupPayloadVersion3{
		MinorVersion: 0,
		Parameters: []packets.StartupPayloadVersion3PayloadParameter{
			{
				Key:   "user",
				Value: username,
			},
			{
				Key:   "database",
				Value: ctx.Options.Database,
			},
		},
	}

	for key, value := range ctx.Options.StartupParameters {
		m.Parameters = append(m.Parameters, packets.StartupPayloadVersion3PayloadParameter{
			Key:   key.String(),
			Value: value,
		})
	}

	p := packets.Startup{
		Mode: &m,
	}

	err := ctx.Conn.WritePacket(&p)
	if err != nil {
		return err
	}

	for {
		var done bool
		done, err = startup0(ctx)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	for {
		var done bool
		done, err = startup1(ctx)
		if err != nil {
			return err
		}
		if done {
			break
		}
	}

	// startup complete, connection is ready for queries
	return nil
}

func Accept(
	conn *fed.Conn,
	sslMode bouncer.SSLMode,
	sslConfig *tls.Config,
	username string,
	credentials auth.Credentials,
	database string,
	startupParameters map[strutil.CIString]string,
) error {
	ctx := acceptContext{
		Conn: conn,
		Options: acceptOptions{
			SSLMode:           sslMode,
			SSLConfig:         sslConfig,
			Username:          username,
			Credentials:       credentials,
			Database:          database,
			StartupParameters: startupParameters,
		},
	}
	return accept(&ctx)
}
