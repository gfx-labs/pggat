package zap

type Reader interface {
	ReadByte() (byte, error)
	Read(*Packet) error
	ReadUntyped(*UntypedPacket) error

	Close() error
}
