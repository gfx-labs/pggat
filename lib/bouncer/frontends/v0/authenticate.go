package frontends

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"io"
	"time"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/perror"
)

type DBAuthenticator struct {
	tracer trace.Tracer
}

type authParams struct {
	Conn    *fed.Conn
	Options authOptions
}

func NewDBAuthenticator() *DBAuthenticator {
	return &DBAuthenticator{
		tracer: otel.Tracer("postgres authenticator", trace.WithInstrumentationAttributes(
			attribute.String("component", "gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0/authenticate.go"),
		)),
	}
}

func authenticationSASLInitial(ctx context.Context, params *authParams, creds auth.SASLServer) (tool auth.SASLVerifier, resp []byte, done bool, err error) {
	// check which authentication method the client wants
	var packet fed.Packet
	packet, err = params.Conn.ReadPacket(ctx, true)
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
			err = nil
			return
		}
		return
	}
	return
}

func authenticationSASLContinue(ctx context.Context, params *authParams, tool auth.SASLVerifier) (resp []byte, done bool, err error) {
	var packet fed.Packet
	packet, err = params.Conn.ReadPacket(ctx, true)
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
			err = nil
			return
		}
		return
	}
	return
}

func (T *DBAuthenticator) authenticationSASL(ctx context.Context, params *authParams, creds auth.SASLServer) error {
	ctx, span := T.tracer.Start(ctx, "authenticationSASL", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

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
	err := params.Conn.WritePacket(ctx, &saslInitial)
	if err != nil {
		return err
	}

	tool, resp, done, err := authenticationSASLInitial(ctx, params, creds)
	if err != nil {
		return err
	}

	for {
		if done {
			m := packets.AuthenticationPayloadSASLFinal(resp)
			final := packets.Authentication{
				Mode: &m,
			}
			err = params.Conn.WritePacket(ctx, &final)
			if err != nil {
				return err
			}
			break
		} else {
			m := packets.AuthenticationPayloadSASLContinue(resp)
			cont := packets.Authentication{
				Mode: &m,
			}
			err = params.Conn.WritePacket(ctx, &cont)
			if err != nil {
				return err
			}
		}

		resp, done, err = authenticationSASLContinue(ctx, params, tool)
		if err != nil {
			return err
		}
	}

	return nil
}

func (T *DBAuthenticator) authenticationMD5(ctx context.Context, params *authParams, creds auth.MD5Server) error {
	ctx, span := T.tracer.Start(ctx, "authenticationMD5", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()

	var salt [4]byte
	_, err := rand.Read(salt[:])
	if err != nil {
		return err
	}
	mode := packets.AuthenticationPayloadMD5Password(salt)
	md5Initial := packets.Authentication{
		Mode: &mode,
	}

	err = params.Conn.WritePacket(ctx, &md5Initial)
	if err != nil {
		return err
	}

	var packet fed.Packet
	packet, err = params.Conn.ReadPacket(ctx, true)
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

func (T *DBAuthenticator) authenticate(ctx context.Context, params *authParams) (err error) {
	if params.Options.Credentials != nil {
		if credsSASL, ok := params.Options.Credentials.(auth.SASLServer); ok {
			err = T.authenticationSASL(ctx, params, credsSASL)
		} else if credsMD5, ok := params.Options.Credentials.(auth.MD5Server); ok {
			err = T.authenticationMD5(ctx, params, credsMD5)
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
	if err = params.Conn.WritePacket(ctx, &authOk); err != nil {
		return
	}
	params.Conn.Authenticated = true

	// send backend key data
	var processID [4]byte
	if _, err = rand.Reader.Read(processID[:]); err != nil {
		return
	}
	var backendKey [4]byte
	if _, err = rand.Reader.Read(backendKey[:]); err != nil {
		return
	}
	params.Conn.BackendKey = fed.BackendKey{
		ProcessID: int32(binary.BigEndian.Uint32(processID[:])),
		SecretKey: int32(binary.BigEndian.Uint32(backendKey[:])),
	}

	keyData := packets.BackendKeyData{
		ProcessID: params.Conn.BackendKey.ProcessID,
		SecretKey: params.Conn.BackendKey.SecretKey,
	}
	if err = params.Conn.WritePacket(ctx, &keyData); err != nil {
		return
	}

	return
}

func (T *DBAuthenticator) Authenticate(ctx context.Context, conn *fed.Conn, creds auth.Credentials) (err error) {
	if conn.Authenticated {
		// already authenticated
		return
	}

	func() {
		ctx, span := T.tracer.Start(ctx, "authenticate", trace.WithSpanKind(trace.SpanKindInternal))
		defer span.End()

		params := authParams{
			Conn: conn,
			Options: authOptions{
				Credentials: creds,
			},
		}
		err = T.authenticate(ctx, &params)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
	}()

	if err != nil {
		// sleep after incorrect password
		time.Sleep(250 * time.Millisecond)
	}
	return
}

func Authenticate(ctx context.Context, conn *fed.Conn, creds auth.Credentials) (err error) {
	return NewDBAuthenticator().Authenticate(ctx, conn, creds)
}
