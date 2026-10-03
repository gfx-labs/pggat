package netconncodec

// HasBufferedTypedPacket reports whether the whole next typed packet is already buffered.
func (c *Codec) HasBufferedTypedPacket() bool {
	return c.decoder.HasBufferedTypedPacket()
}
