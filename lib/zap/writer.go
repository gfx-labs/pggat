package zap

import (
	"io"
)

type Writer interface {
	WriteByte(byte) error
	Write(*Packet) error
	WriteUntyped(*UntypedPacket) error
	WriteV(*Packets) error

	Close() error
}

func WrapIOWriter(writeCloser io.WriteCloser) Writer {
	return ioWriter{
		writer: writeCloser,
		closer: writeCloser,
	}
}

type ioWriter struct {
	writer io.Writer
	closer io.Closer
}

func (T ioWriter) WriteByte(b byte) error {
	_, err := T.writer.Write([]byte{b})
	return err
}

func (T ioWriter) Write(packet *Packet) error {
	_, err := packet.WriteTo(T.writer)
	return err
}

func (T ioWriter) WriteUntyped(packet *UntypedPacket) error {
	_, err := packet.WriteTo(T.writer)
	return err
}

func (T ioWriter) WriteV(packets *Packets) error {
	_, err := packets.WriteTo(T.writer)
	return err
}

func (T ioWriter) Close() error {
	return T.closer.Close()
}

var _ Writer = ioWriter{}
