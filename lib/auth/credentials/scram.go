package credentials

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strconv"
	"strings"
	"sync"

	"gfx.cafe/ghalliday1/scram"

	"gfx.cafe/gfx/pggat/lib/auth"
)

type Scram struct {
	Keys scram.ServerKeys

	clientKey []byte
	mu        sync.RWMutex
}

func ScramFromString(password string) (*Scram, error) {
	alg, iterKeys, ok := strings.Cut(password, "$")
	if !ok {
		return nil, ErrInvalidSecretFormat
	}
	var hasher scram.Hasher
	switch alg {
	case "SCRAM-SHA-256":
		hasher = sha256.New
	default:
		// invalid algorithm
		return nil, ErrInvalidSecretFormat
	}

	iterSalt, keys, ok := strings.Cut(iterKeys, "$")
	if !ok {
		return nil, ErrInvalidSecretFormat
	}
	iter, salt, ok := strings.Cut(iterSalt, ":")
	if !ok {
		return nil, ErrInvalidSecretFormat
	}
	storedKey, serverKey, ok := strings.Cut(keys, ":")

	var res Scram
	res.Keys.Hasher = hasher

	var err error
	res.Keys.Iters, err = strconv.Atoi(iter)
	if err != nil {
		return nil, err
	}

	var saltBytes []byte
	saltBytes, err = base64.StdEncoding.DecodeString(salt)
	if err != nil {
		return nil, err
	}
	res.Keys.Salt = saltBytes

	res.Keys.StoredKey, err = base64.StdEncoding.DecodeString(storedKey)
	if err != nil {
		return nil, err
	}

	res.Keys.ServerKey, err = base64.StdEncoding.DecodeString(serverKey)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

func (T *Scram) SupportedSASLMechanisms() []auth.SASLMechanism {
	return []auth.SASLMechanism{
		auth.ScramSHA256,
	}
}

func (T *Scram) EncodeSASL(mechanisms []auth.SASLMechanism) (auth.SASLMechanism, auth.SASLEncoder, error) {
	T.mu.RLock()
	clientKey := T.clientKey
	T.mu.RUnlock()
	if clientKey == nil {
		return "", nil, errors.New("you must log in with SASL first")
	}

	for _, mechanism := range mechanisms {
		switch mechanism {
		case auth.ScramSHA256:
			return auth.ScramSHA256, &scram.ClientConversation{
				Lookup: scram.ClientKeysLookup(scram.ClientKeys{
					ClientKey: clientKey,
					ServerKey: T.Keys.ServerKey,
					KeyInfo:   T.Keys.KeyInfo,
				}),
			}, nil
		}
	}
	return "", nil, auth.ErrSASLMechanismNotSupported
}

type ScramInterceptorVerifier struct {
	Scram        *Scram
	Conversation *scram.ServerConversation
}

func (T ScramInterceptorVerifier) Write(bytes []byte) ([]byte, error) {
	resp, err := T.Conversation.Write(bytes)
	if err == io.EOF {
		T.Scram.mu.Lock()
		defer T.Scram.mu.Unlock()

		T.Scram.clientKey = T.Conversation.RecoveredClientKey
	}
	return resp, err
}

var _ auth.SASLVerifier = ScramInterceptorVerifier{}

func (T *Scram) VerifySASL(mechanism auth.SASLMechanism) (auth.SASLVerifier, error) {
	switch mechanism {
	case auth.ScramSHA256:
		return ScramInterceptorVerifier{
			Scram: T,
			Conversation: &scram.ServerConversation{
				Lookup: func(string) (scram.ServerKeys, bool) {
					return T.Keys, true
				},
			},
		}, nil
	default:
		return nil, auth.ErrSASLMechanismNotSupported
	}
}

func (*Scram) Credentials() {}

var _ auth.Credentials = (*Scram)(nil)
var _ auth.SASLServer = (*Scram)(nil)
var _ auth.SASLClient = (*Scram)(nil)
