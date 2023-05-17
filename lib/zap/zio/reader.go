package zio

import (
	"io"
	"time"

	"pggat2/lib/util/dio"
	"pggat2/lib/zap"
)

type Reader struct {
	reader dio.Reader
	r      io.Reader
	buf    zap.Buf
}

func MakeReader(reader dio.Reader) Reader {
	return Reader{
		reader: reader,
		r:      reader,
	}
}

func (T *Reader) SetReadDeadline(deadline time.Time) error {
	return T.reader.SetReadDeadline(deadline)
}

func (T *Reader) ReadByte() (byte, error) {
	return T.buf.ReadByte(T.r)
}

func (T *Reader) Read() (zap.In, error) {
	return T.buf.Read(T.r, true)
}

func (T *Reader) ReadUntyped() (zap.In, error) {
	return T.buf.Read(T.r, false)
}

func (T *Reader) Done() {
	T.buf.Done()
}

var _ zap.Reader = (*Reader)(nil)
