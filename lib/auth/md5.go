package auth

import (
	"crypto/md5"
	"encoding/hex"
	"strings"
)

func EncodeMD5(username, password string, salt [4]byte) string {
	hash := md5.New()
	hash.Write([]byte(password))
	hash.Write([]byte(username))
	sum1 := hash.Sum(nil)
	hexEncoded := make([]byte, hex.EncodedLen(len(sum1)))
	hex.Encode(hexEncoded, sum1)
	hash.Reset()

	hash.Write(hexEncoded)
	hash.Write(salt[:])
	sum2 := hash.Sum(nil)
	hexEncoded = hexEncoded[:0]
	for i := 0; i < hex.EncodedLen(len(sum2)); i++ {
		hexEncoded = append(hexEncoded, 0)
	}
	hex.Encode(hexEncoded, sum2)

	var out strings.Builder
	out.Grow(3 + len(hexEncoded))
	out.WriteString("md5")
	out.Write(hexEncoded)
	return out.String()
}

func CheckMD5(username, password string, salt [4]byte, encoded string) bool {
	return EncodeMD5(username, password, salt) == encoded
}
