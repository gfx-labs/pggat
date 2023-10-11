package fed

import (
	"bufio"
	"encoding/binary"
	"io"
	"math"
	"strings"

	"gfx.cafe/gfx/pggat/lib/util/decorator"
)

type Decoder struct {
	noCopy decorator.NoCopy

	Reader bufio.Reader

	typ Type
	len int
	pos int

	buf [8]byte
}

func NewDecoder(r io.Reader) *Decoder {
	d := &Decoder{}
	d.Reader.Reset(r)
	return d
}

func (T *Decoder) ReadByte() (byte, error) {
	if T.pos != T.len {
		_, err := T.Reader.Discard(T.len - T.pos)
		if err != nil {
			return 0, err
		}
	}

	T.typ = 0
	T.len = 0
	T.pos = 0
	return T.Reader.ReadByte()
}

func (T *Decoder) Next(typed bool) error {
	if T.pos != T.len {
		_, err := T.Reader.Discard(T.len - T.pos)
		if err != nil {
			return err
		}
	}

	var err error
	if typed {
		_, err = io.ReadFull(&T.Reader, T.buf[:5])
	} else {
		T.buf[0] = 0
		_, err = io.ReadFull(&T.Reader, T.buf[1:5])
	}
	if err != nil {
		return err
	}
	T.typ = Type(T.buf[0])
	T.len = int(binary.BigEndian.Uint32(T.buf[1:5])) - 4
	T.pos = 0
	return nil
}

func (T *Decoder) Type() Type {
	return T.typ
}

func (T *Decoder) Length() int {
	return T.len
}

func (T *Decoder) Position() int {
	return T.pos
}

func (T *Decoder) Uint8() (uint8, error) {
	v, err := T.Reader.ReadByte()
	T.pos += 1
	return v, err
}

func (T *Decoder) Uint16() (uint16, error) {
	_, err := io.ReadFull(&T.Reader, T.buf[:2])
	T.pos += 2
	return binary.BigEndian.Uint16(T.buf[:2]), err
}

func (T *Decoder) Uint32() (uint32, error) {
	_, err := io.ReadFull(&T.Reader, T.buf[:4])
	T.pos += 4
	return binary.BigEndian.Uint32(T.buf[:4]), err
}

func (T *Decoder) Uint64() (uint64, error) {
	_, err := io.ReadFull(&T.Reader, T.buf[:8])
	T.pos += 8
	return binary.BigEndian.Uint64(T.buf[:8]), err
}

func (T *Decoder) Int8() (int8, error) {
	v, err := T.Uint8()
	return int8(v), err
}

func (T *Decoder) Int16() (int16, error) {
	v, err := T.Uint16()
	return int16(v), err
}

func (T *Decoder) Int32() (int32, error) {
	v, err := T.Uint32()
	return int32(v), err
}

func (T *Decoder) Int64() (int64, error) {
	v, err := T.Uint64()
	return int64(v), err
}

func (T *Decoder) Float32() (float32, error) {
	v, err := T.Uint32()
	return math.Float32frombits(v), err
}

func (T *Decoder) Float64() (float64, error) {
	v, err := T.Uint64()
	return math.Float64frombits(v), err
}

func (T *Decoder) String() (string, error) {
	var s strings.Builder
	for {
		b, err := T.Reader.ReadByte()
		T.pos += 1
		if err != nil {
			return "", err
		}
		if b == '\x00' {
			break
		} else {
			s.WriteByte(b)
		}
	}
	return s.String(), nil
}

func (T *Decoder) Remaining() ([]byte, error) {
	b := make([]byte, T.len-T.pos)
	err := T.Bytes(b)
	return b, err
}

func (T *Decoder) Bytes(b []byte) error {
	n, err := io.ReadFull(&T.Reader, b)
	T.pos += n
	return err
}
