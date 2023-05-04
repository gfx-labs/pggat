package packet

import (
	"encoding/binary"
	"math"
)

type In struct {
	noCopy noCopy
	typ    Type
	buf    []byte
	pos    int
	done   bool
	finish func([]byte)
}

func MakeIn(
	typ Type,
	buf []byte,
	finish func([]byte),
) In {
	return In{
		typ:    typ,
		buf:    buf,
		finish: finish,
	}
}

func (T *In) Type() Type {
	return T.typ
}

// Full returns the full payload of the packet.
// NOTE: Full will be invalid after Done is called
func (T *In) Full() []byte {
	if T.done {
		panic("Read after Done")
	}
	return T.buf
}

// Remaining returns the remaining payload of the packet.
// NOTE: Remaining will be invalid after Done is called
func (T *In) Remaining() []byte {
	full := T.Full()
	return full[T.pos:]
}

func (T *In) Int8() (int8, bool) {
	v, ok := T.Uint8()
	return int8(v), ok
}

func (T *In) Int16() (int16, bool) {
	v, ok := T.Uint16()
	return int16(v), ok
}

func (T *In) Int32() (int32, bool) {
	v, ok := T.Uint32()
	return int32(v), ok
}

func (T *In) Int64() (int64, bool) {
	v, ok := T.Uint64()
	return int64(v), ok
}

func (T *In) Uint8() (uint8, bool) {
	rem := T.Remaining()
	if len(rem) < 1 {
		return 0, false
	}
	v := rem[0]
	T.pos += 1
	return v, true
}

func (T *In) Uint16() (uint16, bool) {
	rem := T.Remaining()
	if len(rem) < 2 {
		return 0, false
	}
	v := binary.BigEndian.Uint16(rem)
	T.pos += 2
	return v, true
}

func (T *In) Uint32() (uint32, bool) {
	rem := T.Remaining()
	if len(rem) < 4 {
		return 0, false
	}
	v := binary.BigEndian.Uint32(rem)
	T.pos += 4
	return v, true
}

func (T *In) Uint64() (uint64, bool) {
	rem := T.Remaining()
	if len(rem) < 8 {
		return 0, false
	}
	v := binary.BigEndian.Uint64(rem)
	T.pos += 8
	return v, true
}

func (T *In) Float32() (float32, bool) {
	v, ok := T.Uint32()
	return math.Float32frombits(v), ok
}

func (T *In) Float64() (float64, bool) {
	v, ok := T.Uint64()
	return math.Float64frombits(v), ok
}

func (T *In) String() (string, bool) {
	rem := T.Remaining()
	for i, c := range rem {
		if c == 0 {
			v := string(rem[:i])
			T.pos += i + 1
			return v, true
		}
	}
	return "", false
}

func (T *In) Bytes(b []byte) bool {
	rem := T.Remaining()
	if len(b) > len(rem) {
		return false
	}
	copy(b, rem)
	T.pos += len(b)
	return true
}

func (T *In) Done() {
	if T.done {
		panic("Done called twice")
	}
	T.done = true
	T.finish(T.buf)
}
