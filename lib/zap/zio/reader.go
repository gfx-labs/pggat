package zio

import (
	"io"

	"pggat2/lib/zap"
)

type Reader struct {
	reader io.Reader
	buf    zap.Buf
}

func MakeReader(reader io.Reader) Reader {
	return Reader{
		reader: reader,
	}
}

func (T *Reader) ReadByte() (byte, error) {
	return T.buf.ReadByte(T.reader)
}

func (T *Reader) Read() (zap.In, error) {
	return T.buf.Read(T.reader, true)
}

func (T *Reader) ReadUntyped() (zap.In, error) {
	return T.buf.Read(T.reader, false)
}

var _ zap.Reader = (*Reader)(nil)
