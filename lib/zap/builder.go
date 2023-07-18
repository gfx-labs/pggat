package zap

import (
	"encoding/binary"
	"math"

	"pggat2/lib/util/slices"
)

type Builder struct {
	buffer *Buffer
	offset int
	typed  bool
}

func (T *Builder) Header() []byte {
	if T.typed {
		return T.buffer.primary[T.offset : T.offset+5]
	}
	return T.buffer.primary[T.offset : T.offset+4]
}

func (T *Builder) Payload() []byte {
	length := T.GetLength()
	if T.typed {
		return T.buffer.primary[T.offset+5 : T.offset+5+length]
	}
	return T.buffer.primary[T.offset+4 : T.offset+4+length]
}

func (T *Builder) next(n int) []byte {
	// add n to length
	oldLength := T.GetLength()
	T.Length(oldLength + n)

	if T.typed {
		return T.buffer.primary[T.offset+oldLength+1 : T.offset+oldLength+n+1]
	}
	return T.buffer.primary[T.offset+oldLength : T.offset+oldLength+n]
}

func (T *Builder) Reset() {
	T.Length(4)
	if T.typed {
		T.Type(0)
	}
}

func (T *Builder) Length(n int) {
	header := T.Header()
	length := header[len(header)-4:]

	binary.BigEndian.PutUint32(length, uint32(n))

	if T.typed {
		T.buffer.primary = slices.Resize(T.buffer.primary, T.offset+n+1)
	} else {
		T.buffer.primary = slices.Resize(T.buffer.primary, T.offset+n)
	}
}

func (T *Builder) GetLength() int {
	header := T.Header()
	length := header[len(header)-4:]

	return int(binary.BigEndian.Uint32(length))
}

func (T *Builder) Type(typ Type) {
	if !T.typed {
		panic("Type() called on untyped builder")
	}

	T.buffer.primary[T.offset] = typ
}

func (T *Builder) Int8(v int8) {
	T.Uint8(uint8(v))
}

func (T *Builder) Int16(v int16) {
	T.Uint16(uint16(v))
}

func (T *Builder) Int32(v int32) {
	T.Uint32(uint32(v))
}

func (T *Builder) Int64(v int64) {
	T.Uint64(uint64(v))
}

func (T *Builder) Uint8(v uint8) {
	T.next(1)[0] = v
}

func (T *Builder) Uint16(v uint16) {
	binary.BigEndian.PutUint16(T.next(2), v)
}

func (T *Builder) Uint32(v uint32) {
	binary.BigEndian.PutUint32(T.next(4), v)
}

func (T *Builder) Uint64(v uint64) {
	binary.BigEndian.PutUint64(T.next(8), v)
}

func (T *Builder) Float32(v float32) {
	T.Uint32(math.Float32bits(v))
}

func (T *Builder) Float64(v float64) {
	T.Uint64(math.Float64bits(v))
}

func (T *Builder) String(v string) {
	copy(T.next(len(v)), v)
	T.Uint8(0)
}

func (T *Builder) Bytes(v []byte) {
	copy(T.next(len(v)), v)
}
