package zap

import "time"

type Reader interface {
	ReadInto(buffer *Buffer, typed bool) error

	SetReadDeadline(time time.Time) error
}
