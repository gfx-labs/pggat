package zap

import (
	"io"
	"time"
)

type Writer interface {
	io.ByteWriter

	SetWriteDeadline(deadline time.Time) error
	Write() Out
	Send(Out) error
}
