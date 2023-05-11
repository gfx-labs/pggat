package zap

import "io"

type Writer interface {
	io.ByteWriter

	Write() Out
	Send(Out) error
}
