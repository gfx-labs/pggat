package credentials

import (
	"encoding/hex"
	"strings"

	"pggat/lib/auth"
)

func FromString(user, password string) auth.Credentials {
	if password == "" {
		return nil
	} else if strings.HasPrefix(password, "md5") {
		hexHash := strings.TrimPrefix(password, "md5")
		hash, err := hex.DecodeString(hexHash)
		if err != nil {
			return Cleartext{
				Username: user,
				Password: password,
			}
		}
		return MD5{
			Username: user,
			Hash:     hash,
		}
	} else {
		return Cleartext{
			Username: user,
			Password: password, // TODO(garet) sasl
		}
	}
}
