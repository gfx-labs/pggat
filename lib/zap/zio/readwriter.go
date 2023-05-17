package zio

import (
	"io"
	"time"

	"pggat2/lib/util/dio"
	"pggat2/lib/zap"
)

type ReadWriter struct {
	rw dio.ReadWriter
	// they are seperated out to prevent an expensive runtime.convI2I (which causes runtime.getitab)
	r   io.Reader
	w   io.Writer
	buf zap.Buf
}

func MakeReadWriter(rw dio.ReadWriter) ReadWriter {
	return ReadWriter{
		rw: rw,
		r:  rw,
		w:  rw,
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
	return T.buf.ReadByte(T.r)
}

func (T *ReadWriter) Read() (zap.In, error) {
	return T.buf.Read(T.r, true)
}

func (T *ReadWriter) ReadUntyped() (zap.In, error) {
	return T.buf.Read(T.r, false)
}

func (T *ReadWriter) WriteByte(b byte) error {
	return T.buf.WriteByte(T.w, b)
}

func (T *ReadWriter) Write() zap.Out {
	return T.buf.Write()
}

func (T *ReadWriter) Send(out zap.Out) error {
	_, err := T.rw.Write(out.Full())
	return err
}

func (T *ReadWriter) Done() {
	T.buf.Done()
}

var _ zap.ReadWriter = (*ReadWriter)(nil)
