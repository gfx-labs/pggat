package psql

import (
	"io"

	"pggat2/lib/zap"
)

type packetReader struct {
	packets []zap.Packet
}

func (T *packetReader) ReadByte() (byte, error) {
	return 0, io.EOF
}

func (T *packetReader) ReadPacket(typed bool) (zap.Packet, error) {
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

var _ zap.Reader = (*packetReader)(nil)

type eofReader struct{}

func (eofReader) ReadByte() (byte, error) {
	return 0, io.EOF
}

func (eofReader) ReadPacket(_ bool) (zap.Packet, error) {
	return nil, io.EOF
}

var _ zap.Reader = eofReader{}
