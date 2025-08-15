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

	//nolint:gosec // G304: Reading pgbouncer auth file from configured path is required functionality.
	// The file path is provided via configuration to specify where the pgbouncer-compatible
	// authentication file is located, which is necessary for pgbouncer compatibility mode.
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
