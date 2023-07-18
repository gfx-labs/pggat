package zio

import (
	"io"
	"time"

	"pggat2/lib/util/dio"
	"pggat2/lib/zap"
)

type ReadWriter struct {
	io dio.ReadWriter
}

func MakeReadWriter(io dio.ReadWriter) ReadWriter {
	return ReadWriter{
		io: io,
	}
}

func (T ReadWriter) ReadInto(buffer *zap.Buffer, typed bool) error {
	builder := buffer.Build(typed)
	_, err := io.ReadFull(T.io, builder.Header())
	if err != nil {
		return err
	}

	builder.Length(builder.GetLength())

	_, err = io.ReadFull(T.io, builder.Payload())
	return err
}

func (T ReadWriter) WriteFrom(buffer *zap.Buffer) error {
	_, err := T.io.Write(buffer.Full())
	return err
}

func (T ReadWriter) SetReadDeadline(time time.Time) error {
	return T.io.SetReadDeadline(time)
}

func (T ReadWriter) SetWriteDeadline(time time.Time) error {
	return T.io.SetWriteDeadline(time)
}

var _ zap.ReadWriter = ReadWriter{}
