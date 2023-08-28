package frontends

import (
	"crypto/rand"
	"errors"

	"pggat2/lib/auth"
	"pggat2/lib/perror"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

func authenticationSASLInitial(client zap.Conn, creds auth.SASL) (tool auth.SASLVerifier, resp []byte, done bool, err perror.Error) {
	// check which authentication method the client wants
	packet, err2 := client.ReadPacket(true)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}
	var initialResponse packets.SASLInitialResponse
	if !initialResponse.ReadFromPacket(packet) {
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
		if errors.Is(err2, auth.ErrSASLComplete) {
			done = true
			return
		}
		err = perror.Wrap(err2)
		return
	}
	return
}

func authenticationSASLContinue(client zap.Conn, tool auth.SASLVerifier) (resp []byte, done bool, err perror.Error) {
	packet, err2 := client.ReadPacket(true)
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}
	var authResp packets.AuthenticationResponse
	if !authResp.ReadFromPacket(packet) {
		err = packets.ErrBadFormat
		return
	}

	resp, err2 = tool.Write(authResp)
	if err2 != nil {
		if errors.Is(err2, auth.ErrSASLComplete) {
			done = true
			return
		}
		err = perror.Wrap(err2)
		return
	}
	return
}

func authenticationSASL(client zap.Conn, creds auth.SASL) perror.Error {
	saslInitial := packets.AuthenticationSASL{
		Mechanisms: creds.SupportedSASLMechanisms(),
	}
	err := perror.Wrap(client.WritePacket(saslInitial.IntoPacket()))
	if err != nil {
		return err
	}

	tool, resp, done, err := authenticationSASLInitial(client, creds)
	if err != nil {
		return err
	}

	for {
		if done {
			final := packets.AuthenticationSASLFinal(resp)
			err = perror.Wrap(client.WritePacket(final.IntoPacket()))
			if err != nil {
				return err
			}
			break
		} else {
			cont := packets.AuthenticationSASLContinue(resp)
			err = perror.Wrap(client.WritePacket(cont.IntoPacket()))
			if err != nil {
				return err
			}
		}

		resp, done, err = authenticationSASLContinue(client, tool)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateParameter(client zap.Conn, name, value string) perror.Error {
	ps := packets.ParameterStatus{
		Key:   name,
		Value: value,
	}
	return perror.Wrap(client.WritePacket(ps.IntoPacket()))
}

func authenticate(client zap.Conn, options AuthenticateOptions) (params AuthenticateParams, err perror.Error) {
	if options.Credentials == nil {
		err = perror.New(
			perror.FATAL,
			perror.InvalidPassword,
			"User or database not found",
		)
		return
	}
	if credsSASL, ok := options.Credentials.(auth.SASL); ok {
		err = authenticationSASL(client, credsSASL)
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

	// send auth Ok
	authOk := packets.AuthenticationOk{}
	if err = perror.Wrap(client.WritePacket(authOk.IntoPacket())); err != nil {
		return
	}

	// send backend key data
	_, err2 := rand.Read(params.BackendKey[:])
	if err2 != nil {
		err = perror.Wrap(err2)
		return
	}

	keyData := packets.BackendKeyData{
		CancellationKey: params.BackendKey,
	}
	if err = perror.Wrap(client.WritePacket(keyData.IntoPacket())); err != nil {
		return
	}

	if err = updateParameter(client, "client_encoding", "UTF8"); err != nil {
		return
	}
	if err = updateParameter(client, "server_encoding", "UTF8"); err != nil {
		return
	}
	if err = updateParameter(client, "server_version", "14.5"); err != nil {
		return
	}

	// send ready for query
	rfq := packets.ReadyForQuery('I')
	if err = perror.Wrap(client.WritePacket(rfq.IntoPacket())); err != nil {
		return
	}

	return
}

func Authenticate(client zap.Conn, options AuthenticateOptions) (AuthenticateParams, perror.Error) {
	params, err := authenticate(client, options)
	if err != nil {
		fail(client, err)
		return AuthenticateParams{}, err
	}
	return params, nil
}
