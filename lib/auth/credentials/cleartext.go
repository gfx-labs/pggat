package credentials

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"github.com/xdg-go/scram"

	"pggat/lib/auth"
	"pggat/lib/util/slices"
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
	hash := md5.New()
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

func (T Cleartext) VerifySASL(mechanism auth.SASLMechanism) (auth.SASLVerifier, error) {
	switch mechanism {
	case auth.ScramSHA256:
		return MakeCleartextScramVerifier(T.Username, T.Password, scram.SHA256)
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
