package dio

import "time"

type ReadWriter interface {
	SetDeadline(deadline time.Time) error
	Reader
	Writer
}
