package zap3

import "io"

type Reader interface {
	Read(*Packet) error
	ReadUntyped(*UntypedPacket) error
}

type IOReader struct {
	Reader io.Reader
}

func (T IOReader) Read(packet *Packet) error {
	_, err := packet.ReadFrom(T.Reader)
	return err
}

func (T IOReader) ReadUntyped(packet *UntypedPacket) error {
	_, err := packet.ReadFrom(T.Reader)
	return err
}

var _ Reader = IOReader{}
