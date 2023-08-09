package credentials

import (
	"crypto/md5"
	"encoding/hex"
	"strings"

	"pggat2/lib/auth"
	"pggat2/lib/util/slices"
)

type MD5 struct {
	Username string
	Hash     []byte
}

func (T MD5) GetUsername() string {
	return T.Username
}

func (T MD5) EncodeMD5(salt [4]byte) string {
	hexEncoded := make([]byte, hex.EncodedLen(len(T.Hash)))
	hex.Encode(hexEncoded, T.Hash)
	hash := md5.New()

	hash.Write(hexEncoded)
	hash.Write(salt[:])
	sum := hash.Sum(nil)
	hexEncoded = slices.Resize(hexEncoded, hex.EncodedLen(len(sum)))
	hex.Encode(hexEncoded, sum)

	var out strings.Builder
	out.Grow(3 + len(hexEncoded))
	out.WriteString("md5")
	out.Write(hexEncoded)
	return out.String()
}

func (T MD5) VerifyMD5(salt [4]byte, value string) error {
	if T.EncodeMD5(salt) != value {
		return auth.ErrFailed
	}

	return nil
}

var _ auth.MD5 = MD5{}
