package packet

import (
	"encoding/binary"
	"math"

	"pggat2/lib/util/decorator"
)

type InBuf struct {
	noCopy decorator.NoCopy
	typ    Type
	buf    []byte
	pos    int
	rev    int
}

func (T *InBuf) Reset(
	typ Type,
	buf []byte,
) {
	T.typ = typ
	T.buf = buf
	T.pos = 0
	T.rev++
}

type In struct {
	buf *InBuf
	rev int
}

func MakeIn(
	buf *InBuf,
) In {
	return In{
		buf: buf,
		rev: buf.rev,
	}
}

func (T In) done() bool {
	return T.rev != T.buf.rev
}

func (T In) Type() Type {
	if T.done() {
		panic("Read after Send")
	}
	return T.buf.typ
}

// Full returns the full payload of the packet.
// NOTE: Full will be invalid after Done is called
func (T In) Full() []byte {
	if T.done() {
		panic("Read after Send")
	}
	return T.buf.buf
}

// Remaining returns the remaining payload of the packet.
// NOTE: Remaining will be invalid after Done is called
func (T In) Remaining() []byte {
	full := T.Full()
	return full[T.buf.pos:]
}

func (T In) Reset() {
	if T.done() {
		panic("Read after Send")
	}
	T.buf.pos = 0
}

func (T In) Int8() (int8, bool) {
	v, ok := T.Uint8()
	return int8(v), ok
}

func (T In) Int16() (int16, bool) {
	v, ok := T.Uint16()
	return int16(v), ok
}

func (T In) Int32() (int32, bool) {
	v, ok := T.Uint32()
	return int32(v), ok
}

func (T In) Int64() (int64, bool) {
	v, ok := T.Uint64()
	return int64(v), ok
}

func (T In) Uint8() (uint8, bool) {
	rem := T.Remaining()
	if len(rem) < 1 {
		return 0, false
	}
	v := rem[0]
	T.buf.pos += 1
	return v, true
}

func (T In) Uint16() (uint16, bool) {
	rem := T.Remaining()
	if len(rem) < 2 {
		return 0, false
	}
	v := binary.BigEndian.Uint16(rem)
	T.buf.pos += 2
	return v, true
}

func (T In) Uint32() (uint32, bool) {
	rem := T.Remaining()
	if len(rem) < 4 {
		return 0, false
	}
	v := binary.BigEndian.Uint32(rem)
	T.buf.pos += 4
	return v, true
}

func (T In) Uint64() (uint64, bool) {
	rem := T.Remaining()
	if len(rem) < 8 {
		return 0, false
	}
	v := binary.BigEndian.Uint64(rem)
	T.buf.pos += 8
	return v, true
}

func (T In) Float32() (float32, bool) {
	v, ok := T.Uint32()
	return math.Float32frombits(v), ok
}

func (T In) Float64() (float64, bool) {
	v, ok := T.Uint64()
	return math.Float64frombits(v), ok
}

func (T In) String() (string, bool) {
	rem := T.Remaining()
	for i, c := range rem {
		if c == 0 {
			v := string(rem[:i])
			T.buf.pos += i + 1
			return v, true
		}
	}
	return "", false
}

func (T In) Bytes(b []byte) bool {
	rem := T.Remaining()
	if len(b) > len(rem) {
		return false
	}
	copy(b, rem)
	T.buf.pos += len(b)
	return true
}

// UnsafeBytes returns a byte slice without copying. Use this only if you plan to be done with the slice when the In is reset or the data will be replaced with garbage.
func (T In) UnsafeBytes(count int) ([]byte, bool) {
	rem := T.Remaining()
	if count > len(rem) {
		return nil, false
	}
	return rem[:count], true
}
