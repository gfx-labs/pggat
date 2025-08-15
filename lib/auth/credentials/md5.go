package credentials

import (
	"crypto/md5" //nolint:gosec // MD5 required for PostgreSQL authentication protocol
	"encoding/hex"
	"strings"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type MD5 struct {
	Hash []byte
}

func MD5FromString(value string) (MD5, error) {
	if !strings.HasPrefix(value, "md5") {
		return MD5{}, ErrInvalidSecretFormat
	}

	var res MD5
	var err error
	hexString := strings.TrimPrefix(value, "md5")
	res.Hash, err = hex.DecodeString(hexString)
	if err != nil {
		return MD5{}, err
	}

	return res, nil
}

func (MD5) Credentials() {}

func (T MD5) EncodeMD5(salt [4]byte) string {
	hexEncoded := make([]byte, hex.EncodedLen(len(T.Hash)))
	hex.Encode(hexEncoded, T.Hash)
	hash := md5.New() //nolint:gosec // MD5 required for PostgreSQL authentication protocol

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

var _ auth.Credentials = MD5{}
var _ auth.MD5Client = MD5{}
var _ auth.MD5Server = MD5{}
