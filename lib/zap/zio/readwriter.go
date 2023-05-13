package zio

import (
	"time"

	"pggat2/lib/util/dio"
	"pggat2/lib/zap"
)

type ReadWriter struct {
	rw  dio.ReadWriter
	buf zap.Buf
}

func MakeReadWriter(rw dio.ReadWriter) ReadWriter {
	return ReadWriter{
		rw: rw,
	}
}

func (T *ReadWriter) SetDeadline(deadline time.Time) error {
	return T.rw.SetDeadline(deadline)
}

func (T *ReadWriter) SetReadDeadline(deadline time.Time) error {
	return T.rw.SetReadDeadline(deadline)
}

func (T *ReadWriter) SetWriteDeadline(deadline time.Time) error {
	return T.rw.SetWriteDeadline(deadline)
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
