package frontends

import (
	"crypto/rand"
	"errors"
	"io"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
)

func authenticationSASLInitial(ctx *authenticateContext, creds auth.SASLServer) (tool auth.SASLVerifier, resp []byte, done bool, err perror.Error) {
	// check which authentication method the client wants
	var err2 error
	ctx.Packet, err2 = ctx.Conn.ReadPacket(true, ctx.Packet)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}
	var initialResponse packets.SASLInitialResponse
	if !initialResponse.ReadFromPacket(ctx.Packet) {
		err = packets.ErrBadFormat
		return
	}

	tool, err2 = creds.VerifySASL(initialResponse.Mechanism)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}

	resp, err2 = tool.Write(initialResponse.InitialResponse)
	if err2 != nil {
		if errors.Is(err2, io.EOF) {
			done = true
			return
		}
		err = perror.Wrap(err2)
		return
	}
	return
}

func authenticationSASLContinue(ctx *authenticateContext, tool auth.SASLVerifier) (resp []byte, done bool, err perror.Error) {
	var err2 error
	ctx.Packet, err2 = ctx.Conn.ReadPacket(true, ctx.Packet)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}
	var authResp packets.AuthenticationResponse
	if !authResp.ReadFromPacket(ctx.Packet) {
		err = packets.ErrBadFormat
		return
	}

	resp, err2 = tool.Write(authResp)
	if err2 != nil {
		if errors.Is(err2, io.EOF) {
			done = true
			return
		}
		err = perror.Wrap(err2)
		return
	}
	return
}

func authenticationSASL(ctx *authenticateContext, creds auth.SASLServer) perror.Error {
	saslInitial := packets.AuthenticationSASL{
		Mechanisms: creds.SupportedSASLMechanisms(),
	}
	ctx.Packet = saslInitial.IntoPacket(ctx.Packet)
	err := perror.Wrap(ctx.Conn.WritePacket(ctx.Packet))
	if err != nil {
		return err
	}

	tool, resp, done, err := authenticationSASLInitial(ctx, creds)
	if err != nil {
		return err
	}

	for {
		if done {
			final := packets.AuthenticationSASLFinal(resp)
			ctx.Packet = final.IntoPacket(ctx.Packet)
			err = perror.Wrap(ctx.Conn.WritePacket(ctx.Packet))
			if err != nil {
				return err
			}
			break
		} else {
			cont := packets.AuthenticationSASLContinue(resp)
			ctx.Packet = cont.IntoPacket(ctx.Packet)
			err = perror.Wrap(ctx.Conn.WritePacket(ctx.Packet))
			if err != nil {
				return err
			}
		}

		resp, done, err = authenticationSASLContinue(ctx, tool)
		if err != nil {
			return err
		}
	}

	return nil
}

func authenticationMD5(ctx *authenticateContext, creds auth.MD5Server) perror.Error {
	var salt [4]byte
	_, err := rand.Read(salt[:])
	if err != nil {
		return perror.Wrap(err)
	}
	md5Initial := packets.AuthenticationMD5{
		Salt: salt,
	}
	ctx.Packet = md5Initial.IntoPacket(ctx.Packet)
	err = ctx.Conn.WritePacket(ctx.Packet)
	if err != nil {
		return perror.Wrap(err)
	}

	ctx.Packet, err = ctx.Conn.ReadPacket(true, ctx.Packet)
	if err != nil {
		return perror.Wrap(err)
	}

	var pw packets.PasswordMessage
	if !pw.ReadFromPacket(ctx.Packet) {
		return packets.ErrUnexpectedPacket
	}

	if err = creds.VerifyMD5(salt, pw.Password); err != nil {
		return perror.Wrap(err)
	}

	return nil
}

func authenticate(ctx *authenticateContext) (err perror.Error) {
	if ctx.Options.Credentials != nil {
		if credsSASL, ok := ctx.Options.Credentials.(auth.SASLServer); ok {
			err = authenticationSASL(ctx, credsSASL)
		} else if credsMD5, ok := ctx.Options.Credentials.(auth.MD5Server); ok {
			err = authenticationMD5(ctx, credsMD5)
		} else {
			err = perror.New(
				perror.FATAL,
				perror.InternalError,
				"Auth method not supported",
			)
		}
		if err != nil {
			return
		}
	}

	// send auth Ok
	authOk := packets.AuthenticationOk{}
	ctx.Packet = authOk.IntoPacket(ctx.Packet)
	if err = perror.Wrap(ctx.Conn.WritePacket(ctx.Packet)); err != nil {
		return
	}
	ctx.Conn.Authenticated = true

	// send backend key data
	_, err2 := rand.Read(ctx.Conn.BackendKey[:])
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}

	keyData := packets.BackendKeyData{
		CancellationKey: ctx.Conn.BackendKey,
	}
	ctx.Packet = keyData.IntoPacket(ctx.Packet)
	if err = perror.Wrap(ctx.Conn.WritePacket(ctx.Packet)); err != nil {
		return
	}

	return
}

func Authenticate(conn *fed.Conn, creds auth.Credentials) (err perror.Error) {
	if conn.Authenticated {
		// already authenticated
		return
	}

	ctx := authenticateContext{
		Conn: conn,
		Options: authenticateOptions{
			Credentials: creds,
		},
	}
	err = authenticate(&ctx)
	return
}
