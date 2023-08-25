package bouncer

import (
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
)

type Conn struct {
	RW zap.Conn

	SSLEnabled        bool
	User              string
	Database          string
	InitialParameters map[strutil.CIString]string
	BackendKey        [8]byte
}
