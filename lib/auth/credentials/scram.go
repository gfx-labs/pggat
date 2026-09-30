package credentials

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
	"sync"

	"github.com/gfx-labs/scram"

	"gfx.cafe/gfx/pggat/lib/auth"
)

type Scram struct {
	Keys scram.ServerKeys

	clientKeys *scram.ClientKeys
	mu         sync.RWMutex
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
	if !ok {
		return nil, ErrInvalidSecretFormat
	}

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
	clientKeys := T.clientKeys
	T.mu.RUnlock()
	if clientKeys == nil {
		return "", nil, errors.New("you must log in with SASL first")
	}

	for _, mechanism := range mechanisms {
		if mechanism == auth.ScramSHA256 {
			return auth.ScramSHA256, scram.NewClientConversation(scram.ClientConfig{
				Lookup: scram.ClientKeysLookup(*clientKeys),
			}), nil
		}
	}
	return "", nil, auth.ErrSASLMechanismNotSupported
}

// ScramInterceptorVerifier records the client keys recovered from a successful
// exchange so the same credentials can authenticate to upstream servers.
type ScramInterceptorVerifier struct {
	Scram        *Scram
	Conversation *scram.ServerConversation
}

func (T ScramInterceptorVerifier) Step(in []byte) ([]byte, error) {
	resp, err := T.Conversation.Step(in)
	if err != nil {
		return resp, err
	}
	if T.Conversation.Authenticated() {
		keys, err := T.Conversation.ClientKeys()
		if err != nil {
			return nil, err
		}
		T.Scram.mu.Lock()
		T.Scram.clientKeys = &keys
		T.Scram.mu.Unlock()
	}
	return resp, nil
}

func (T ScramInterceptorVerifier) Done() bool {
	return T.Conversation.Done()
}

var _ auth.SASLVerifier = ScramInterceptorVerifier{}

func (T *Scram) VerifySASL(mechanism auth.SASLMechanism) (auth.SASLVerifier, error) {
	switch mechanism {
	case auth.ScramSHA256:
		return ScramInterceptorVerifier{
			Scram: T,
			Conversation: scram.NewServerConversation(&scram.ServerConfig{
				Lookup: func(string) (scram.ServerKeys, error) {
					return T.Keys, nil
				},
			}),
		}, nil
	default:
		return nil, auth.ErrSASLMechanismNotSupported
	}
}

func (*Scram) Credentials() {}

var _ auth.Credentials = (*Scram)(nil)
var _ auth.SASLServer = (*Scram)(nil)
var _ auth.SASLClient = (*Scram)(nil)
