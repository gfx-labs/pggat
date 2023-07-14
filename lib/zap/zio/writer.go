package zio

import (
	"io"
	"time"

	"pggat2/lib/util/dio"
	"pggat2/lib/zap"
)

type Writer struct {
	writer dio.Writer
	w      io.Writer
	buf    zap.Buf
}

func MakeWriter(writer dio.Writer) Writer {
	return Writer{
		writer: writer,
		w:      writer,
	}
}

func (T *Writer) SetWriteDeadline(deadline time.Time) error {
	return T.writer.SetWriteDeadline(deadline)
}

func (T *Writer) WriteByte(b byte) error {
	return T.buf.WriteByte(T.w, b)
}

func (T *Writer) Write() zap.Out {
	return T.buf.Write()
}

func (T *Writer) Send(out zap.Out) error {
	_, err := T.writer.Write(out.Full())
	return err
}

func (T *Writer) Done() {
	T.buf.Done()
}

var _ zap.Writer = (*Writer)(nil)
