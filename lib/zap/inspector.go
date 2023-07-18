package zap

import (
	"encoding/binary"
	"math"
)

type Inspector struct {
	buffer *Buffer
	offset int
	typed  bool

	position int
}

func (T *Inspector) Reset() {
	if T.typed {
		T.position = 5
	} else {
		T.position = 4
	}
}

func (T *Inspector) Length() int {
	var length []byte
	if T.typed {
		length = T.buffer.primary[T.offset+1 : T.offset+5]
	} else {
		length = T.buffer.primary[T.offset : T.offset+4]
	}

	return int(binary.BigEndian.Uint32(length))
}

func (T *Inspector) Payload() []byte {
	length := T.Length()
	if T.typed {
		return T.buffer.primary[T.offset+5 : T.offset+5+length]
	}
	return T.buffer.primary[T.offset+4 : T.offset+4+length]
}

func (T *Inspector) Remaining() []byte {
	if T.typed {
		return T.buffer.primary[T.offset+T.position : T.offset+T.Length()+1]
	} else {
		return T.buffer.primary[T.offset+T.position : T.offset+T.Length()]
	}
}

func (T *Inspector) Type() Type {
	if !T.typed {
		panic("call of Type() on untyped packet")
	}

	return T.buffer.primary[T.offset]
}

func (T *Inspector) Int8() (int8, bool) {
	if v, ok := T.Uint8(); ok {
		return int8(v), true
	}
	return 0, false
}

func (T *Inspector) Int16() (int16, bool) {
	if v, ok := T.Uint16(); ok {
		return int16(v), true
	}
	return 0, false
}

func (T *Inspector) Int32() (int32, bool) {
	if v, ok := T.Uint32(); ok {
		return int32(v), true
	}
	return 0, false
}

func (T *Inspector) Int64() (int64, bool) {
	if v, ok := T.Uint64(); ok {
		return int64(v), true
	}
	return 0, false
}

func (T *Inspector) Uint8() (uint8, bool) {
	rem := T.Remaining()
	if len(rem) < 1 {
		return 0, false
	}
	T.position += 1
	return rem[0], true
}

func (T *Inspector) Uint16() (uint16, bool) {
	rem := T.Remaining()
	if len(rem) < 2 {
		return 0, false
	}
	T.position += 2
	return binary.BigEndian.Uint16(rem), true
}

func (T *Inspector) Uint32() (uint32, bool) {
	rem := T.Remaining()
	if len(rem) < 4 {
		return 0, false
	}
	T.position += 4
	return binary.BigEndian.Uint32(rem), true
}

func (T *Inspector) Uint64() (uint64, bool) {
	rem := T.Remaining()
	if len(rem) < 8 {
		return 0, false
	}
	T.position += 8
	return binary.BigEndian.Uint64(rem), true
}

func (T *Inspector) Float32() (float32, bool) {
	if v, ok := T.Uint32(); ok {
		return math.Float32frombits(v), true
	}
	return 0, false
}

func (T *Inspector) Float64() (float64, bool) {
	if v, ok := T.Uint64(); ok {
		return math.Float64frombits(v), true
	}
	return 0, false
}

func (T *Inspector) String() (string, bool) {
	rem := T.Remaining()
	for i, c := range rem {
		if c == 0 {
			T.position += i + 1
			return string(rem[:i]), true
		}
	}
	return "", false
}

func (T *Inspector) Bytes(b []byte) bool {
	rem := T.Remaining()
	if len(rem) < len(b) {
		return false
	}
	T.position += copy(b, rem)
	return true
}

func (T *Inspector) UnsafeBytes(count int) ([]byte, bool) {
	rem := T.Remaining()
	if len(rem) < count {
		return nil, false
	}
	T.position += count
	return rem[:count], true
}
