package pnet

import "io"

type IOReadWriter struct {
	IOReader
	IOWriter
}

func MakeIOReadWriter(inner io.ReadWriter) IOReadWriter {
	return IOReadWriter{
		IOReader: MakeIOReader(inner),
		IOWriter: MakeIOWriter(inner),
	}
}

func NewIOReadWriter(inner io.ReadWriter) *IOReadWriter {
	rw := MakeIOReadWriter(inner)
	return &rw
}

var _ ReadWriter = (*IOReadWriter)(nil)
