package session

import (
	"github.com/google/uuid"

	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
)

type Conn struct {
	id                uuid.UUID
	rw                zap.ReadWriter
	initialParameters map[strutil.CIString]string
}
