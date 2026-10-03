package fed

import "encoding/binary"

// HasBufferedTypedPacket reports whether the whole next typed packet, header and body, is already
// in the read buffer, so Next and reading its body cannot block. It is conservative: a packet larger
// than the buffer or an invalid length reports false.
func (T *Decoder) HasBufferedTypedPacket() bool {
	rem := T.packetLength - T.packetPos
	if rem < 0 {
		return false
	}

	buf := T.buffer[T.bufferRead:T.bufferWrite]
	if rem > len(buf) {
		return false
	}
	buf = buf[rem:]

	if len(buf) < 5 {
		return false
	}
	// the length field counts itself and excludes the type byte
	length := binary.BigEndian.Uint32(buf[1:5])
	if length < 4 {
		return false
	}
	return uint64(len(buf)-5) >= uint64(length-4)
}
