package fed

// bufferedPacketReader is implemented by codecs that can tell whether the next packet is fully buffered.
type bufferedPacketReader interface {
	HasBufferedTypedPacket() bool
}

// HasBufferedPacket reports whether the next typed packet can be read without blocking on the
// transport. A relay uses it to flush its output only before a read that may block. Codecs that cannot
// tell report false, which makes callers flush conservatively.
func (T *Conn) HasBufferedPacket() bool {
	if c, ok := T.codec.(bufferedPacketReader); ok {
		return c.HasBufferedTypedPacket()
	}
	return false
}
