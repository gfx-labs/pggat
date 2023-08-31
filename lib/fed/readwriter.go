package fed

type Reader interface {
	ReadByte() (byte, error)
	ReadPacket(typed bool) (Packet, error)
}

type Writer interface {
	WriteByte(byte) error
	WritePacket(Packet) error
}

type ReadWriter interface {
	Reader
	Writer
}

type CombinedReadWriter struct {
	Reader
	Writer
}
