package pgbouncer

import (
	"os"

	"gfx.cafe/gfx/pggat/lib/util/encoding/ini"
	"gfx.cafe/gfx/pggat/lib/util/encoding/userlist"
)

type AuthFile map[string]string

func (T *AuthFile) UnmarshalINI(bytes []byte) error {
	path := string(bytes)
	if path == "" {
		return nil
	}

	file, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	*T, err = userlist.Unmarshal(file)
	if err != nil {
		return err
	}

	return nil
}

var _ ini.Unmarshaller = (*AuthFile)(nil)
