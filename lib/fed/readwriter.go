package fed

type Reader interface {
	ReadPacket(typed bool, buffer Packet) (Packet, error)
}

type Writer interface {
	WritePacket(Packet) error
}

type ReadWriter interface {
	Reader
	Writer
}

type ReadWriteCloser interface {
	ReadWriter

	Close() error
}
