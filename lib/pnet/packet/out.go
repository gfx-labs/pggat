package packet

import (
	"encoding/binary"
	"math"
)

type Out struct {
	noCopy noCopy
	typ    Type
	buf    []byte
	done   bool
	finish func(Type, []byte) error
}

func MakeOut(
	buf []byte,
	finish func(Type, []byte) error,
) Out {
	return Out{
		buf:    buf,
		finish: finish,
	}
}

func (T *Out) Type(typ Type) {
	T.typ = typ
}

func (T *Out) Int8(v int8) {
	T.Uint8(uint8(v))
}

func (T *Out) Int16(v int16) {
	T.Uint16(uint16(v))
}

func (T *Out) Int32(v int32) {
	T.Uint32(uint32(v))
}

func (T *Out) Int64(v int64) {
	T.Uint64(uint64(v))
}

func (T *Out) Uint8(v uint8) {
	if T.done {
		panic("Write after Done")
	}
	T.buf = append(T.buf, v)
}

func (T *Out) Uint16(v uint16) {
	if T.done {
		panic("Write after Done")
	}
	T.buf = binary.BigEndian.AppendUint16(T.buf, v)
}

func (T *Out) Uint32(v uint32) {
	if T.done {
		panic("Write after Done")
	}
	T.buf = binary.BigEndian.AppendUint32(T.buf, v)
}

func (T *Out) Uint64(v uint64) {
	if T.done {
		panic("Write after Done")
	}
	T.buf = binary.BigEndian.AppendUint64(T.buf, v)
}

func (T *Out) Float32(v float32) {
	T.Uint32(math.Float32bits(v))
}

func (T *Out) Float64(v float64) {
	T.Uint64(math.Float64bits(v))
}

func (T *Out) String(v string) {
	if T.done {
		panic("Write after Done")
	}
	T.buf = append(T.buf, v...)
	T.Uint8(0)
}

func (T *Out) Bytes(v []byte) {
	if T.done {
		panic("Write after Done")
	}
	T.buf = append(T.buf, v...)
}

func (T *Out) Done() error {
	if T.done {
		panic("Done called twice")
	}
	T.done = true
	return T.finish(T.typ, T.buf)
}
