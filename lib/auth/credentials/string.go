package credentials

import (
	"pggat/lib/auth"
)

func FromString(user, password string) auth.Credentials {
	if password == "" {
		return nil
	} else if v, err := ScramFromString(user, password); err == nil {
		return v
	} else if v, err := MD5FromString(password); err == nil {
		return v
	} else {
		return Cleartext{
			Username: user,
			Password: password,
		}
	}
}
