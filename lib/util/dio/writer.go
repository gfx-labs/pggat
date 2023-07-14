package dio

import (
	"io"
	"time"
)

type Writer interface {
	SetWriteDeadline(deadline time.Time) error
	io.Writer
}
