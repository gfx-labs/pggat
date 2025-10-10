package credentials

import (
	"crypto/md5" //nolint:gosec // MD5 required for PostgreSQL authentication protocol
	"crypto/rand"
	"encoding/hex"
	"strings"

	"github.com/minio/sha256-simd"

	"gfx.cafe/ghalliday1/scram"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Cleartext struct {
	Username string
	Password string
}

func (Cleartext) Credentials() {}

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
	hash := md5.New() //nolint:gosec // MD5 required for PostgreSQL authentication protocol
	hash.Write([]byte(T.Password))
	hash.Write([]byte(T.Username))
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

func (T Cleartext) EncodeSASL(mechanisms []auth.SASLMechanism) (auth.SASLMechanism, auth.SASLEncoder, error) {
	for _, mechanism := range mechanisms {
		if mechanism == auth.ScramSHA256 {
			return auth.ScramSHA256, &scram.ClientConversation{
				Lookup: scram.ClientPasswordLookup(T.Password, sha256.New),
			}, nil
		}
	}
	return "", nil, auth.ErrSASLMechanismNotSupported
}

func (T Cleartext) VerifySASL(mechanism auth.SASLMechanism) (auth.SASLVerifier, error) {
	switch mechanism {
	case auth.ScramSHA256:
		return &scram.ServerConversation{
			Lookup: func(string) (scram.ServerKeys, bool) {
				var salt [32]byte
				_, err := rand.Read(salt[:])
				if err != nil {
					return scram.ServerKeys{}, false
				}
				hasher := scram.Hasher(sha256.New)
				keyInfo := scram.KeyInfo{
					Salt:   salt[:],
					Iters:  4096,
					Hasher: hasher,
				}
				saltedPassword := hasher.SaltedPassword([]byte(T.Password), keyInfo.Salt, keyInfo.Iters)
				serverKey := hasher.ServerKey(saltedPassword)
				clientKey := hasher.ClientKey(saltedPassword)
				storedKey := hasher.StoredKey(clientKey)

				return scram.ServerKeys{
					ServerKey: serverKey,
					StoredKey: storedKey,
					KeyInfo:   keyInfo,
				}, true
			},
		}, nil
	default:
		return nil, auth.ErrSASLMechanismNotSupported
	}
}

var _ auth.Credentials = Cleartext{}
var _ auth.CleartextClient = Cleartext{}
var _ auth.CleartextServer = Cleartext{}
var _ auth.MD5Client = Cleartext{}
var _ auth.MD5Server = Cleartext{}
var _ auth.SASLClient = Cleartext{}
var _ auth.SASLServer = Cleartext{}
