package zap

import "io"

type ReadWriter interface {
	io.ByteReader
	io.ByteWriter
	io.Closer

	EnableSSL(client bool) error

	Read(*Packet) error
	ReadUntyped(*UntypedPacket) error
	Write(*Packet) error
	WriteUntyped(*UntypedPacket) error
	WriteV(*Packets) error
}
