package fed

import "context"

// Middleware intercepts packets and possibly changes them. Return a 0 length packet to cancel.
type Middleware interface {
	PreRead(ctx context.Context,typed bool) (Packet, error)
	ReadPacket(ctx context.Context,packet Packet) (Packet, error)

	WritePacket(ctx context.Context,packet Packet) (Packet, error)
	PostWrite(ctx context.Context,) (Packet, error)
}

func LookupMiddleware[T Middleware](conn *Conn) (T, bool) {
	for _, mw := range conn.Middleware {
		m, ok := mw.(T)
		if ok {
			return m, true
		}
	}

	return *new(T), false
}
