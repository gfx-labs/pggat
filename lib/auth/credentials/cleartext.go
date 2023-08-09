package credentials

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"github.com/xdg-go/scram"

	"pggat2/lib/auth"
	"pggat2/lib/util/slices"
)

type Cleartext struct {
	Username string
	Password string
}

func (T Cleartext) GetUsername() string {
	return T.Username
}

func (T Cleartext) EncodeCleartext() string {
	return T.Password
}

func (T Cleartext) VerifyCleartext(value string) error {
	if T.Password != value {
		return auth.ErrFailed
	}
	return nil
}

func (T Cleartext) EncodeMD5(salt [4]byte) string {
	hash := md5.New()
	hash.Write([]byte(T.Username))
	hash.Write([]byte(T.Password))
	sum1 := hash.Sum(nil)
	hexEncoded := make([]byte, hex.EncodedLen(len(sum1)))
	hex.Encode(hexEncoded, sum1)
	hash.Reset()

	hash.Write(hexEncoded)
	hash.Write(salt[:])
	sum2 := hash.Sum(nil)
	hexEncoded = slices.Resize(hexEncoded, hex.EncodedLen(len(sum2)))
	hex.Encode(hexEncoded, sum2)

	var out strings.Builder
	out.Grow(3 + len(hexEncoded))
	out.WriteString("md5")
	out.Write(hexEncoded)
	return out.String()
}

func (T Cleartext) VerifyMD5(salt [4]byte, value string) error {
	if T.EncodeMD5(salt) != value {
		return auth.ErrFailed
	}

	return nil
}

func (T Cleartext) SupportedSASLMechanisms() []auth.SASLMechanism {
	return []auth.SASLMechanism{
		auth.ScramSHA256,
	}
}

type CleartextScramEncoder struct {
	conversation *scram.ClientConversation
}

func MakeCleartextScramEncoder(username, password string, hashGenerator scram.HashGeneratorFcn) (CleartextScramEncoder, error) {
	client, err := hashGenerator.NewClient(username, password, "")
	if err != nil {
		return CleartextScramEncoder{}, err
	}

	return CleartextScramEncoder{
		conversation: client.NewConversation(),
	}, nil
}

func (T CleartextScramEncoder) Write(bytes []byte) ([]byte, error) {
	if bytes == nil {
		// initial response
		return nil, nil
	}

	msg, err := T.conversation.Step(string(bytes))
	if err != nil {
		return nil, err
	}
	return []byte(msg), nil
}

var _ auth.SASLEncoder = CleartextScramEncoder{}

func (T Cleartext) EncodeSASL(mechanisms []auth.SASLMechanism) (auth.SASLMechanism, auth.SASLEncoder, error) {
	for _, mechanism := range mechanisms {
		switch mechanism {
		case auth.ScramSHA256:
			encoder, err := MakeCleartextScramEncoder(T.Username, T.Password, scram.SHA256)
			if err != nil {
				return "", nil, err
			}

			return auth.ScramSHA256, encoder, nil
		}
	}
	return "", nil, auth.ErrSASLMechanismNotSupported
}

type CleartextScramVerifier struct {
	conversation *scram.ServerConversation
}

func MakeCleartextScramVerifier(username, password string, hashGenerator scram.HashGeneratorFcn) (CleartextScramVerifier, error) {
	client, err := hashGenerator.NewClient(username, password, "")
	if err != nil {
		return CleartextScramVerifier{}, err
	}

	kf := scram.KeyFactors{
		Iters: 1,
	}
	stored := client.GetStoredCredentials(kf)

	server, err := hashGenerator.NewServer(
		func(string) (scram.StoredCredentials, error) {
			return stored, nil
		},
	)
	if err != nil {
		return CleartextScramVerifier{}, err
	}

	return CleartextScramVerifier{
		conversation: server.NewConversation(),
	}, nil
}

func (T CleartextScramVerifier) Write(bytes []byte) ([]byte, error) {
	msg, err := T.conversation.Step(string(bytes))
	if err != nil {
		return nil, err
	}

	if T.conversation.Done() {
		// check if conversation params are valid
		if !T.conversation.Valid() {
			return nil, auth.ErrFailed
		}

		// done
		return []byte(msg), auth.ErrSASLComplete
	}

	// there is more
	return []byte(msg), nil
}

var _ auth.SASLVerifier = CleartextScramVerifier{}

func (T Cleartext) VerifySASL(mechanism auth.SASLMechanism) (auth.SASLVerifier, error) {
	switch mechanism {
	case auth.ScramSHA256:
		return MakeCleartextScramVerifier(T.Username, T.Password, scram.SHA256)
	default:
		return nil, auth.ErrSASLMechanismNotSupported
	}
}

var _ auth.Credentials = Cleartext{}
var _ auth.Cleartext = Cleartext{}
var _ auth.MD5 = Cleartext{}
var _ auth.SASL = Cleartext{}
