package packet

import (
	"encoding/binary"
	"math"
)

type Reader struct {
	typ   Type
	bytes []byte
}

func MakeReader(raw Raw) Reader {
	return Reader{
		typ:   raw.Type,
		bytes: raw.Payload,
	}
}

func (T *Reader) Type() Type {
	return T.typ
}

func (T *Reader) Int8() (int8, bool) {
	v, ok := T.Uint8()
	return int8(v), ok
}

func (T *Reader) Int16() (int16, bool) {
	v, ok := T.Uint16()
	return int16(v), ok
}

func (T *Reader) Int32() (int32, bool) {
	v, ok := T.Uint32()
	return int32(v), ok
}

func (T *Reader) Int64() (int64, bool) {
	v, ok := T.Uint64()
	return int64(v), ok
}

func (T *Reader) Uint8() (uint8, bool) {
	if len(T.bytes) < 1 {
		return 0, false
	}
	v := T.bytes[0]
	T.bytes = T.bytes[1:]
	return v, true
}

func (T *Reader) Uint16() (uint16, bool) {
	if len(T.bytes) < 2 {
		return 0, false
	}
	v := binary.BigEndian.Uint16(T.bytes)
	T.bytes = T.bytes[2:]
	return v, true
}

func (T *Reader) Uint32() (uint32, bool) {
	if len(T.bytes) < 4 {
		return 0, false
	}
	v := binary.BigEndian.Uint32(T.bytes)
	T.bytes = T.bytes[4:]
	return v, true
}

func (T *Reader) Uint64() (uint64, bool) {
	if len(T.bytes) < 8 {
		return 0, false
	}
	v := binary.BigEndian.Uint64(T.bytes)
	T.bytes = T.bytes[8:]
	return v, true
}

func (T *Reader) Float32() (float32, bool) {
	v, ok := T.Uint32()
	return math.Float32frombits(v), ok
}

func (T *Reader) Float64() (float64, bool) {
	v, ok := T.Uint64()
	return math.Float64frombits(v), ok
}

func (T *Reader) String() (string, bool) {
	// read up until zero byte
	for i, b := range T.bytes {
		if b == 0 {
			v := string(T.bytes[:i])
			T.bytes = T.bytes[i+1:]
			return v, true
		}
	}
	return "", false
}

func (T *Reader) Bytes(count int) ([]byte, bool) {
	if len(T.bytes) < count {
		return nil, false
	}
	v := T.bytes[:count]
	T.bytes = T.bytes[count:]
	return v, true
}

func (T *Reader) Remaining() []byte {
	return T.bytes
}
