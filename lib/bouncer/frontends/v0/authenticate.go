package frontends

import (
	"crypto/rand"
	"errors"

	"pggat/lib/auth"
	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/perror"
)

func authenticationSASLInitial(client fed.Conn, creds auth.SASL) (tool auth.SASLVerifier, resp []byte, done bool, err perror.Error) {
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

func authenticationSASLContinue(client fed.Conn, tool auth.SASLVerifier) (resp []byte, done bool, err perror.Error) {
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

func authenticationSASL(client fed.Conn, creds auth.SASL) perror.Error {
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

func authenticationMD5(client fed.Conn, creds auth.MD5) perror.Error {
	var salt [4]byte
	_, err := rand.Read(salt[:])
	if err != nil {
		return perror.Wrap(err)
	}
	md5Initial := packets.AuthenticationMD5{
		Salt: salt,
	}
	err = client.WritePacket(md5Initial.IntoPacket())
	if err != nil {
		return perror.Wrap(err)
	}

	var packet fed.Packet
	packet, err = client.ReadPacket(true)
	if err != nil {
		return perror.Wrap(err)
	}

	var pw packets.PasswordMessage
	if !pw.ReadFromPacket(packet) {
		return packets.ErrUnexpectedPacket
	}

	if err = creds.VerifyMD5(salt, pw.Password); err != nil {
		return perror.Wrap(err)
	}

	return nil
}

func authenticate(client fed.Conn, options AuthenticateOptions) (params AuthenticateParams, err perror.Error) {
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
	} else if credsMD5, ok := options.Credentials.(auth.MD5); ok {
		err = authenticationMD5(client, credsMD5)
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

	return
}

func Authenticate(client fed.Conn, options AuthenticateOptions) (AuthenticateParams, perror.Error) {
	params, err := authenticate(client, options)
	if err != nil {
		fail(client, err)
		return AuthenticateParams{}, err
	}
	return params, nil
}
