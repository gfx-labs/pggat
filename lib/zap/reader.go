package zap

import "io"

type Reader interface {
	ReadByte() (byte, error)
	Read(*Packet) error
	ReadUntyped(*UntypedPacket) error
}

type IOReader struct {
	Reader io.Reader
}

func (T IOReader) ReadByte() (byte, error) {
	var res = []byte{0}
	_, err := io.ReadFull(T.Reader, res)
	if err != nil {
		return 0, err
	}
	return res[0], err
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
