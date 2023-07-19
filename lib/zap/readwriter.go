package zap

import "io"

type ReadWriter interface {
	Reader
	Writer
}

type CombinedReadWriter struct {
	Reader
	Writer
}

func WrapIOReadWriter(readWriteCloser io.ReadWriteCloser) ReadWriter {
	return CombinedReadWriter{
		Reader: WrapIOReader(readWriteCloser),
		Writer: WrapIOWriter(readWriteCloser),
	}
}
