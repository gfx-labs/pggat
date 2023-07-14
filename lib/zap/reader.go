package zap

import (
	"io"
	"time"
)

type Reader interface {
	io.ByteReader

	SetReadDeadline(deadline time.Time) error
	Read() (In, error)
	ReadUntyped() (In, error)
}
