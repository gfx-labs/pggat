package frontends

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
)

func authenticationSASLInitial(ctx *authenticateContext, creds auth.SASLServer) (tool auth.SASLVerifier, resp []byte, done bool, err error) {
	// check which authentication method the client wants
	var packet fed.Packet
	packet, err = ctx.Conn.ReadPacket(true)
	if err != nil {
		return
	}
	var p packets.SASLInitialResponse
	err = fed.ToConcrete(&p, packet)
	if err != nil {
		return
	}

	tool, err = creds.VerifySASL(p.Mechanism)
	if err != nil {
		return
	}

	resp, err = tool.Write(p.InitialClientResponse)
	if err != nil {
		if errors.Is(err, io.EOF) {
			done = true
			return
		}
		return
	}
	return
}

func authenticationSASLContinue(ctx *authenticateContext, tool auth.SASLVerifier) (resp []byte, done bool, err error) {
	var packet fed.Packet
	packet, err = ctx.Conn.ReadPacket(true)
	if err != nil {
		return
	}
	var p packets.SASLResponse
	err = fed.ToConcrete(&p, packet)
	if err != nil {
		return
	}

	resp, err = tool.Write(p)
	if err != nil {
		if errors.Is(err, io.EOF) {
			done = true
			return
		}
		return
	}
	return
}

func authenticationSASL(ctx *authenticateContext, creds auth.SASLServer) error {
	var mode packets.AuthenticationPayloadSASL
	mechanisms := creds.SupportedSASLMechanisms()
	for _, mechanism := range mechanisms {
		mode = append(mode, packets.AuthenticationPayloadSASLMethod{
			Method: mechanism,
		})
	}

	saslInitial := packets.Authentication{
		Mode: &mode,
	}
	err := ctx.Conn.WritePacket(&saslInitial)
	if err != nil {
		return err
	}

	tool, resp, done, err := authenticationSASLInitial(ctx, creds)
	if err != nil {
		return err
	}

	for {
		if done {
			m := packets.AuthenticationPayloadSASLFinal(resp)
			final := packets.Authentication{
				Mode: &m,
			}
			err = ctx.Conn.WritePacket(&final)
			if err != nil {
				return err
			}
			break
		} else {
			m := packets.AuthenticationPayloadSASLContinue(resp)
			cont := packets.Authentication{
				Mode: &m,
			}
			err = ctx.Conn.WritePacket(&cont)
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

func authenticationMD5(ctx *authenticateContext, creds auth.MD5Server) error {
	var salt [4]byte
	_, err := rand.Read(salt[:])
	if err != nil {
		return err
	}
	mode := packets.AuthenticationPayloadMD5Password(salt)
	md5Initial := packets.Authentication{
		Mode: &mode,
	}
	err = ctx.Conn.WritePacket(&md5Initial)
	if err != nil {
		return err
	}

	var packet fed.Packet
	packet, err = ctx.Conn.ReadPacket(true)
	if err != nil {
		return err
	}

	var pw packets.PasswordMessage
	err = fed.ToConcrete(&pw, packet)
	if err != nil {
		return err
	}

	if err = creds.VerifyMD5(salt, string(pw)); err != nil {
		return err
	}

	return nil
}

func authenticate(ctx *authenticateContext) (err error) {
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
	authOk := packets.Authentication{
		Mode: &packets.AuthenticationPayloadOk{},
	}
	if err = ctx.Conn.WritePacket(&authOk); err != nil {
		return
	}
	ctx.Conn.Authenticated = true

	// send backend key data
	var processID [4]byte
	if _, err = rand.Reader.Read(processID[:]); err != nil {
		return
	}
	var backendKey [4]byte
	if _, err = rand.Reader.Read(backendKey[:]); err != nil {
		return
	}
	ctx.Conn.BackendKey = fed.BackendKey{
		ProcessID: int32(binary.BigEndian.Uint32(processID[:])),
		SecretKey: int32(binary.BigEndian.Uint32(backendKey[:])),
	}

	keyData := packets.BackendKeyData{
		ProcessID: ctx.Conn.BackendKey.ProcessID,
		SecretKey: ctx.Conn.BackendKey.SecretKey,
	}
	if err = ctx.Conn.WritePacket(&keyData); err != nil {
		return
	}

	return
}

func Authenticate(conn *fed.Conn, creds auth.Credentials) (err error) {
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
