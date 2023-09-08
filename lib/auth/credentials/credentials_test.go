package credentials

import (
	"crypto/rand"
	"testing"

	"pggat/lib/auth"
)

func TestMD5(t *testing.T) {
	pw := FromString("bob", "jNKuKKlBDO48qbLiVw7IuoaamZ1SmHAUdQ9PKH7qRzsyJVF0BNPSFMbHTQwxe0HJ")
	md5 := FromString("bob", "md5e20510fd38e1c0fd99db13da5c29bd95")

	pwMD5 := pw.(auth.MD5)
	md5MD5 := md5.(auth.MD5)

	var salt [4]byte
	_, err := rand.Read(salt[:])
	if err != nil {
		t.Error(err)
		return
	}

	err = md5MD5.VerifyMD5(salt, pwMD5.EncodeMD5(salt))
	if err != nil {
		t.Error(err)
	}
}
