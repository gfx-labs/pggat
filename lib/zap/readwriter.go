package zap

import "crypto/tls"

type ReadWriter interface {
	EnableSSLClient(config *tls.Config) error
	EnableSSLServer(config *tls.Config) error

	ReadByte() (byte, error)
	ReadPacket(typed bool) (Packet, error)

	WriteByte(byte) error
	WritePacket(Packet) error

	Close() error
}
