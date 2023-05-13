package dio

import (
	"io"
	"time"
)

type Reader interface {
	SetReadDeadline(deadline time.Time) error
	io.Reader
}
