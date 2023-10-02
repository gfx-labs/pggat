package fed

// Middleware intercepts packets and possibly changes them. Return a 0 length packet to cancel.
type Middleware interface {
	ReadPacket(packet Packet) (Packet, error)
	WritePacket(packet Packet) (Packet, error)
}
