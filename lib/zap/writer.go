package zap

type Writer interface {
	WriteByte(byte) error
	Write(*Packet) error
	WriteUntyped(*UntypedPacket) error
	WriteV(*Packets) error

	Close() error
}
