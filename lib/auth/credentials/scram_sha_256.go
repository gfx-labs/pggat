package credentials

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/xdg-go/scram"

	"pggat/lib/auth"
)

type ScramSHA256 struct {
	StoredCredentials scram.StoredCredentials
}

func ScramSHA256FromString(value string) (ScramSHA256, error) {
	alg, iterKeys, ok := strings.Cut(value, "$")
	if !ok || alg != "SCRAM-SHA-256" {
		return ScramSHA256{}, ErrInvalidSecretFormat
	}
	iterSalt, keys, ok := strings.Cut(iterKeys, "$")
	if !ok {
		return ScramSHA256{}, ErrInvalidSecretFormat
	}
	iter, salt, ok := strings.Cut(iterSalt, ":")
	if !ok {
		return ScramSHA256{}, ErrInvalidSecretFormat
	}
	storedKey, serverKey, ok := strings.Cut(keys, ":")

	var res ScramSHA256
	var err error
	res.StoredCredentials.Iters, err = strconv.Atoi(iter)
	if err != nil {
		return ScramSHA256{}, err
	}

	var saltBytes []byte
	saltBytes, err = base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return ScramSHA256{}, err
	}
	res.StoredCredentials.Salt = string(saltBytes)

	res.StoredCredentials.StoredKey, err = base64.StdEncoding.DecodeString(storedKey)
	if err != nil {
		return ScramSHA256{}, err
	}

	res.StoredCredentials.ServerKey, err = base64.StdEncoding.DecodeString(serverKey)
	if err != nil {
		return ScramSHA256{}, err
	}

	return res, nil
}

func (T ScramSHA256) SupportedSASLMechanisms() []auth.SASLMechanism {
	return []auth.SASLMechanism{
		auth.ScramSHA256,
	}
}

func (T ScramSHA256) VerifySASL(mechanism auth.SASLMechanism) (auth.SASLVerifier, error) {
	switch mechanism {
	case auth.ScramSHA256:
		return MakeStoredCredentialsScramVerifier(T.StoredCredentials, scram.SHA256)
	default:
		return nil, auth.ErrSASLMechanismNotSupported
	}
}

func (ScramSHA256) Credentials() {}

var _ auth.Credentials = ScramSHA256{}
var _ auth.SASLServer = ScramSHA256{}
