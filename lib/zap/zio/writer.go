package zio

import (
	"io"

	"pggat2/lib/zap"
)

type Writer struct {
	writer io.Writer
	buf    zap.Buf
}

func MakeWriter(writer io.Writer) Writer {
	return Writer{
		writer: writer,
	}
}

func (T *Writer) WriteByte(b byte) error {
	return T.buf.WriteByte(T.writer, b)
}

func (T *Writer) Write() zap.Out {
	return T.buf.Write()
}

func (T *Writer) Send(out zap.Out) error {
	_, err := T.writer.Write(out.Full())
	return err
}

var _ zap.Writer = (*Writer)(nil)
