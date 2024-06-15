package fed

// Middleware intercepts packets and possibly changes them. Return a 0 length packet to cancel.
type Middleware interface {
	PreRead(typed bool) (Packet, error)
	ReadPacket(packet Packet) (Packet, error)

	WritePacket(packet Packet) (Packet, error)
	PostWrite() (Packet, error)
}

func LookupMiddleware[T Middleware](conn Conn) (T, bool) {
	for _, mw := range conn.Middleware() {
		m, ok := mw.(T)
		if ok {
			return m, true
		}
	}

	return *new(T), false
}
