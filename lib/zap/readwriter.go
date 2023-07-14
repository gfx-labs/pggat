package zap

import "time"

type ReadWriter interface {
	SetDeadline(deadline time.Time) error
	Reader
	Writer
}
