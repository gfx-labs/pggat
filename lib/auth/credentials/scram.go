package credentials

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"strings"

	"gfx.cafe/ghalliday1/scram"

	"pggat/lib/auth"
)

func MakeCleartextScramEncoder(username, password string, hashGenerator scram.Hasher) (auth.SASLEncoder, error) {
	return &scram.ClientConversation{
		User:   username,
		Lookup: scram.ClientPasswordLookup(password, hashGenerator),
	}, nil
}

func MakeCleartextScramVerifier(username, password string, hashGenerator scram.Hasher) (auth.SASLVerifier, error) {
	return &scram.ServerConversation{
		Lookup: func(user string) (scram.ServerKeys, bool) {
			if username != user {
				return scram.ServerKeys{}, false
			}

			var salt [32]byte
			_, err := rand.Read(salt[:])
			if err != nil {
				return scram.ServerKeys{}, false
			}
			keyInfo := scram.KeyInfo{
				Salt:   salt[:],
				Iters:  2048,
				Hasher: hashGenerator,
			}
			saltedPassword := hashGenerator.SaltedPassword([]byte(password), keyInfo.Salt, keyInfo.Iters)
			serverKey := hashGenerator.ServerKey(saltedPassword)
			clientKey := hashGenerator.ClientKey(saltedPassword)
			storedKey := hashGenerator.StoredKey(clientKey)

			return scram.ServerKeys{
				ServerKey: serverKey,
				StoredKey: storedKey,
				KeyInfo:   keyInfo,
			}, true
		},
	}, nil
}

func MakeStoredCredentialsScramVerifier(username string, keys scram.ServerKeys) (auth.SASLVerifier, error) {
	return &scram.ServerConversation{
		Lookup: func(user string) (scram.ServerKeys, bool) {
			if user != username {
				return scram.ServerKeys{}, false
			}

			return keys, true
		},
	}, nil
}

type Scram struct {
	User string
	Keys scram.ServerKeys
}

func ScramFromString(user, password string) (Scram, error) {
	alg, iterKeys, ok := strings.Cut(password, "$")
	if !ok {
		return Scram{}, ErrInvalidSecretFormat
	}
	var hasher scram.Hasher
	switch alg {
	case "SCRAM-SHA-256":
		hasher = sha256.New
	default:
		// invalid algorithm
		return Scram{}, ErrInvalidSecretFormat
	}

	iterSalt, keys, ok := strings.Cut(iterKeys, "$")
	if !ok {
		return Scram{}, ErrInvalidSecretFormat
	}
	iter, salt, ok := strings.Cut(iterSalt, ":")
	if !ok {
		return Scram{}, ErrInvalidSecretFormat
	}
	storedKey, serverKey, ok := strings.Cut(keys, ":")

	var res Scram
	res.User = user
	res.Keys.Hasher = hasher

	var err error
	res.Keys.Iters, err = strconv.Atoi(iter)
	if err != nil {
		return Scram{}, err
	}

	var saltBytes []byte
	saltBytes, err = base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return Scram{}, err
	}
	res.Keys.Salt = saltBytes

	res.Keys.StoredKey, err = base64.StdEncoding.DecodeString(storedKey)
	if err != nil {
		return Scram{}, err
	}

	res.Keys.ServerKey, err = base64.StdEncoding.DecodeString(serverKey)
	if err != nil {
		return Scram{}, err
	}

	return res, nil
}

func (T Scram) SupportedSASLMechanisms() []auth.SASLMechanism {
	return []auth.SASLMechanism{
		auth.ScramSHA256,
	}
}

func (T Scram) VerifySASL(mechanism auth.SASLMechanism) (auth.SASLVerifier, error) {
	switch mechanism {
	case auth.ScramSHA256:
		return MakeStoredCredentialsScramVerifier(T.User, T.Keys)
	default:
		return nil, auth.ErrSASLMechanismNotSupported
	}
}

func (Scram) Credentials() {}

var _ auth.Credentials = Scram{}
var _ auth.SASLServer = Scram{}
