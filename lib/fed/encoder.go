package fed

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"gfx.cafe/gfx/pggat/lib/util/decorator"
)

type Encoder struct {
	noCopy decorator.NoCopy

	writer bufio.Writer

	typ Type
	len int
	pos int

	buf [8]byte
}

func NewEncoder(w io.Writer) *Encoder {
	e := new(Encoder)
	e.Reset(w)
	return e
}

var (
	ErrWrongNumberOfBytes = errors.New("wrong number of bytes written")
)

func (T *Encoder) Reset(w io.Writer) {
	T.writer.Reset(w)
}

func (T *Encoder) ReadFrom(r io.Reader) (int64, error) {
	return T.writer.ReadFrom(r)
}

func (T *Encoder) Flush() error {
	return T.writer.Flush()
}

func (T *Encoder) WriteByte(b byte) error {
	if T.pos != T.len {
		return ErrWrongNumberOfBytes
	}

	T.typ = 0
	T.len = 0
	T.pos = 0
	return T.writer.WriteByte(b)
}

func (T *Encoder) Next(typ Type, length int) error {
	if T.pos != T.len {
		return ErrWrongNumberOfBytes
	}

	if typ != 0 {
		if err := T.writer.WriteByte(byte(typ)); err != nil {
			return err
		}
	}

	binary.BigEndian.PutUint32(T.buf[:4], uint32(length+4))
	_, err := T.writer.Write(T.buf[:4])

	T.typ = typ
	T.len = length
	T.pos = 0

	return err
}

func (T *Encoder) Type() Type {
	return T.typ
}

func (T *Encoder) Length() int {
	return T.len
}

func (T *Encoder) Position() int {
	return T.pos
}

func (T *Encoder) Uint8(v uint8) error {
	err := T.writer.WriteByte(v)
	T.pos += 1
	return err
}

func (T *Encoder) Uint16(v uint16) error {
	binary.BigEndian.PutUint16(T.buf[:2], v)
	_, err := T.writer.Write(T.buf[:2])
	T.pos += 2
	return err
}

func (T *Encoder) Uint32(v uint32) error {
	binary.BigEndian.PutUint32(T.buf[:4], v)
	_, err := T.writer.Write(T.buf[:4])
	T.pos += 4
	return err
}

func (T *Encoder) Uint64(v uint64) error {
	binary.BigEndian.PutUint64(T.buf[:8], v)
	_, err := T.writer.Write(T.buf[:8])
	T.pos += 8
	return err
}

func (T *Encoder) Int8(v int8) error {
	return T.Uint8(uint8(v))
}

func (T *Encoder) Int16(v int16) error {
	return T.Uint16(uint16(v))
}

func (T *Encoder) Int32(v int32) error {
	return T.Uint32(uint32(v))
}

func (T *Encoder) Int64(v int64) error {
	return T.Uint64(uint64(v))
}

func (T *Encoder) Float32(v float32) error {
	return T.Uint32(math.Float32bits(v))
}

func (T *Encoder) Float64(v float64) error {
	return T.Uint64(math.Float64bits(v))
}

func (T *Encoder) String(v string) error {
	n, err := T.writer.WriteString(v)
	if err != nil {
		return err
	}
	err = T.writer.WriteByte(0)
	T.pos += n + 1
	return err
}
