package zap

import (
	"io"
)

type Writer interface {
	WriteByte(byte) error
	Write(*Packet) error
	WriteUntyped(*UntypedPacket) error
	WriteV(*Packets) error
}

type IOWriter struct {
	Writer io.Writer
}

func (T IOWriter) WriteByte(b byte) error {
	_, err := T.Writer.Write([]byte{b})
	return err
}

func (T IOWriter) Write(packet *Packet) error {
	_, err := packet.WriteTo(T.Writer)
	return err
}

func (T IOWriter) WriteUntyped(packet *UntypedPacket) error {
	_, err := packet.WriteTo(T.Writer)
	return err
}

func (T IOWriter) WriteV(packets *Packets) error {
	_, err := packets.WriteTo(T.Writer)
	return err
}

var _ Writer = IOWriter{}
