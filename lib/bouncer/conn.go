package bouncer

import (
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
)

type Conn struct {
	RW zap.ReadWriter

	User              string
	Database          string
	InitialParameters map[strutil.CIString]string
	CancellationKey   [8]byte
}
