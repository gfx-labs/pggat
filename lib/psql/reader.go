package psql

import (
	"io"

	"pggat2/lib/fed"
)

type packetReader struct {
	packets []fed.Packet
}

func (T *packetReader) ReadByte() (byte, error) {
	return 0, io.EOF
}

func (T *packetReader) ReadPacket(typed bool) (fed.Packet, error) {
	if len(T.packets) == 0 {
		return nil, io.EOF
	}

	packet := T.packets[0]
	packetTyped := packet.Type() != 0

	if packetTyped != typed {
		return nil, io.EOF
	}

	T.packets = T.packets[1:]
	return packet, nil
}

var _ fed.Reader = (*packetReader)(nil)

type eofReader struct{}

func (eofReader) ReadByte() (byte, error) {
	return 0, io.EOF
}

func (eofReader) ReadPacket(_ bool) (fed.Packet, error) {
	return nil, io.EOF
}

var _ fed.Reader = eofReader{}
