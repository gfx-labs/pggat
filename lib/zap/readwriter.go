package zap

import (
	"crypto/tls"
	"io"
)

type ReadWriter interface {
	io.ByteReader
	io.ByteWriter
	io.Closer

	EnableSSLClient(config *tls.Config) error
	EnableSSLServer(config *tls.Config) error

	Read(*Packet) error
	ReadUntyped(*UntypedPacket) error
	Write(*Packet) error
	WriteUntyped(*UntypedPacket) error
	WriteV(*Packets) error
}
