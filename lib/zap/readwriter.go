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

func (T CombinedReadWriter) Close() error {
	return T.Reader.Close()
}

func WrapIOReadWriter(readWriteCloser io.ReadWriteCloser) ReadWriter {
	return CombinedReadWriter{
		Reader: WrapIOReader(readWriteCloser),
		Writer: WrapIOWriter(readWriteCloser),
	}
}
