package pio

import (
	"io"

	"pggat2/lib/pnet"
)

type ReadWriter struct {
	Reader
	Writer
}

func MakeReadWriter(inner io.ReadWriter) ReadWriter {
	return ReadWriter{
		Reader: MakeReader(inner),
		Writer: MakeWriter(inner),
	}
}

func NewReadWriter(inner io.ReadWriter) *ReadWriter {
	rw := MakeReadWriter(inner)
	return &rw
}

var _ pnet.ReadWriter = (*ReadWriter)(nil)
