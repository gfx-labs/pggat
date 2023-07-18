package zap

import "time"

type Writer interface {
	WriteFrom(buffer *Buffer) error

	SetWriteDeadline(time time.Time) error
}
