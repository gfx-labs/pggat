package fed

import (
	"crypto/tls"
	"net"
)

type PacketCodec interface {
	ReadPacket(typed bool) (Packet, error)
	WritePacket(packet Packet) error
	WriteByte(b byte) error
	ReadByte() (byte, error)

	LocalAddr() net.Addr
	Flush() error
	Close() error

	SSL() bool
	EnableSSL(config *tls.Config, isClient bool) error
}
