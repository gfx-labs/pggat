package zio

import (
	"io"

	"pggat2/lib/zap"
)

type ReadWriter struct {
	rw  io.ReadWriter
	buf zap.Buf
}

func MakeReadWriter(rw io.ReadWriter) ReadWriter {
	return ReadWriter{
		rw: rw,
	}
}

func (T *ReadWriter) ReadByte() (byte, error) {
	return T.buf.ReadByte(T.rw)
}

func (T *ReadWriter) Read() (zap.In, error) {
	return T.buf.Read(T.rw, true)
}

func (T *ReadWriter) ReadUntyped() (zap.In, error) {
	return T.buf.Read(T.rw, false)
}

func (T *ReadWriter) WriteByte(b byte) error {
	return T.buf.WriteByte(T.rw, b)
}

func (T *ReadWriter) Write() zap.Out {
	return T.buf.Write()
}

func (T *ReadWriter) Send(out zap.Out) error {
	_, err := T.rw.Write(out.Full())
	return err
}

var _ zap.ReadWriter = (*ReadWriter)(nil)
