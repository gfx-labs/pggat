package fed

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"math"

	"gfx.cafe/gfx/pggat/lib/util/decorator"
)

type Decoder struct {
	noCopy decorator.NoCopy

	reader bufio.Reader

	typ Type
	len int
	pos int

	buf [8]byte
}

func NewDecoder(r io.Reader) *Decoder {
	d := new(Decoder)
	d.Reset(r)
	return d
}

func (T *Decoder) Reset(r io.Reader) {
	T.reader.Reset(r)
}

func (T *Decoder) Read(b []byte) (n int, err error) {
	rem := T.len - T.pos
	if rem == 0 {
		err = io.EOF
		return
	}
	if len(b) > rem {
		n, err = T.reader.Read(b[:rem])
	} else {
		n, err = T.reader.Read(b)
	}
	T.pos += n
	return
}

func (T *Decoder) Buffered() int {
	return T.reader.Buffered()
}

var ErrOverranReadBuffer = errors.New("overran read buffer")

func (T *Decoder) ReadByte() (byte, error) {
	rem := T.len - T.pos
	if rem < 0 {
		return 0, ErrOverranReadBuffer
	} else if rem > 0 {
		_, err := T.reader.Discard(rem)
		if err != nil {
			return 0, err
		}
	}

	T.typ = 0
	T.len = 0
	T.pos = 0
	return T.reader.ReadByte()
}

func (T *Decoder) Next(typed bool) error {
	rem := T.len - T.pos
	if rem < 0 {
		return ErrOverranReadBuffer
	} else if rem > 0 {
		_, err := T.reader.Discard(rem)
		if err != nil {
			return err
		}
	}

	var err error
	if typed {
		_, err = io.ReadFull(&T.reader, T.buf[:5])
	} else {
		T.buf[0] = 0
		_, err = io.ReadFull(&T.reader, T.buf[1:5])
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
	v, err := T.reader.ReadByte()
	T.pos += 1
	return v, err
}

func (T *Decoder) Uint16() (uint16, error) {
	_, err := io.ReadFull(&T.reader, T.buf[:2])
	T.pos += 2
	return binary.BigEndian.Uint16(T.buf[:2]), err
}

func (T *Decoder) Uint32() (uint32, error) {
	_, err := io.ReadFull(&T.reader, T.buf[:4])
	T.pos += 4
	return binary.BigEndian.Uint32(T.buf[:4]), err
}

func (T *Decoder) Uint64() (uint64, error) {
	_, err := io.ReadFull(&T.reader, T.buf[:8])
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
	s, err := T.reader.ReadString(0)
	if err != nil {
		return "", err
	}
	T.pos += len(s)
	return s[:len(s)-1], nil
}
