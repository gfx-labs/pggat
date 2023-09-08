package pgbouncer

import (
	"os"

	"pggat/lib/util/encoding/ini"
	"pggat/lib/util/encoding/userlist"
)

type AuthFile struct {
	Users map[string]string
}

func (T *AuthFile) UnmarshalINI(bytes []byte) error {
	path := string(bytes)
	if path == "" {
		return nil
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	T.Users, err = userlist.Unmarshal(file)
	if err != nil {
		return err
	}

	return nil
}

var _ ini.Unmarshaller = (*AuthFile)(nil)
