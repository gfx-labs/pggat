package packet

import (
	"encoding/binary"
	"math"

	"pggat2/lib/util/decorator"
)

type OutBuf struct {
	noCopy decorator.NoCopy
	typ    Type
	buf    []byte
	rev    int
}

func (T *OutBuf) Reset() {
	T.typ = None
	T.buf = T.buf[:0]
	T.rev++
}

type Out struct {
	buf *OutBuf
	rev int
}

func MakeOut(
	buf *OutBuf,
) Out {
	return Out{
		buf: buf,
		rev: buf.rev,
	}
}

func (T Out) done() bool {
	return T.rev != T.buf.rev
}

func (T Out) Finish() (Type, []byte) {
	if T.done() {
		panic("Write after Send")
	}
	return T.buf.typ, T.buf.buf
}

func (T Out) Type(typ Type) {
	if T.done() {
		panic("Write after Send")
	}
	T.buf.typ = typ
}

func (T Out) Reset() {
	if T.done() {
		panic("Write after Send")
	}
	T.buf.buf = T.buf.buf[:0]
}

func (T Out) Int8(v int8) {
	T.Uint8(uint8(v))
}

func (T Out) Int16(v int16) {
	T.Uint16(uint16(v))
}

func (T Out) Int32(v int32) {
	T.Uint32(uint32(v))
}

func (T Out) Int64(v int64) {
	T.Uint64(uint64(v))
}

func (T Out) Uint8(v uint8) {
	if T.done() {
		panic("Write after Send")
	}
	T.buf.buf = append(T.buf.buf, v)
}

func (T Out) Uint16(v uint16) {
	if T.done() {
		panic("Write after Send")
	}
	T.buf.buf = binary.BigEndian.AppendUint16(T.buf.buf, v)
}

func (T Out) Uint32(v uint32) {
	if T.done() {
		panic("Write after Send")
	}
	T.buf.buf = binary.BigEndian.AppendUint32(T.buf.buf, v)
}

func (T Out) Uint64(v uint64) {
	if T.done() {
		panic("Write after Send")
	}
	T.buf.buf = binary.BigEndian.AppendUint64(T.buf.buf, v)
}

func (T Out) Float32(v float32) {
	T.Uint32(math.Float32bits(v))
}

func (T Out) Float64(v float64) {
	T.Uint64(math.Float64bits(v))
}

func (T Out) String(v string) {
	if T.done() {
		panic("Write after Send")
	}
	T.buf.buf = append(T.buf.buf, v...)
	T.Uint8(0)
}

func (T Out) Bytes(v []byte) {
	if T.done() {
		panic("Write after Send")
	}
	T.buf.buf = append(T.buf.buf, v...)
}
