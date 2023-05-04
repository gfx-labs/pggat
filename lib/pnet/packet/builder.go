package packet

import (
	"encoding/binary"
	"math"
)

type Builder struct {
	typ   Type
	bytes []byte
}

func MakeBuilder(buf []byte) Builder {
	return Builder{
		bytes: buf,
	}
}

func (T *Builder) Type(typ Type) {
	T.typ = typ
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
	T.bytes = append(T.bytes, v)
}

func (T *Builder) Uint16(v uint16) {
	T.bytes = binary.BigEndian.AppendUint16(T.bytes, v)
}

func (T *Builder) Uint32(v uint32) {
	T.bytes = binary.BigEndian.AppendUint32(T.bytes, v)
}

func (T *Builder) Uint64(v uint64) {
	T.bytes = binary.BigEndian.AppendUint64(T.bytes, v)
}

func (T *Builder) Float32(v float32) {
	T.Uint32(math.Float32bits(v))
}

func (T *Builder) Float64(v float64) {
	T.Uint64(math.Float64bits(v))
}

func (T *Builder) String(v string) {
	T.bytes = append(T.bytes, v...)
	T.bytes = append(T.bytes, 0)
}

func (T *Builder) Bytes(v []byte) {
	T.bytes = append(T.bytes, v...)
}

func (T *Builder) Raw() Raw {
	return Raw{
		Type:    T.typ,
		Payload: T.bytes,
	}
}
