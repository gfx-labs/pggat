package fed

import (
	"encoding/binary"
	"errors"
	"io"
	"math"

	"gfx.cafe/gfx/pggat/lib/util/decorator"
)

const defaultBufferSize = 4 * 1024

type Encoder struct {
	noCopy decorator.NoCopy

	buffer    [defaultBufferSize]byte
	bufferPos int
	writer    io.Writer

	packetType   Type
	packetLength int
	packetPos    int
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		writer: w,
	}
}

var (
	ErrWrongNumberOfBytes = errors.New("wrong number of bytes written")
)

func (T *Encoder) Reset(w io.Writer) {
	T.bufferPos = 0
	T.writer = w
}

func (T *Encoder) ReadFrom(r *Decoder) (int, error) {
	var n int
	for {
		if T.bufferPos >= len(T.buffer) {
			if err := T.Flush(); err != nil {
				T.packetPos += n
				return n, err
			}
		}
		count, err := r.Read(T.buffer[T.bufferPos:])
		T.bufferPos += count
		n += count
		if err == io.EOF {
			break
		}
		if err != nil {
			T.packetPos += n
			return n, err
		}
	}
	T.packetPos += n
	if n == 0 {
		return n, io.ErrUnexpectedEOF
	}
	return n, nil
}

func (T *Encoder) Flush() error {
	if T.bufferPos == 0 {
		return nil
	}
	_, err := T.writer.Write(T.buffer[:T.bufferPos])
	T.bufferPos = 0
	return err
}

func (T *Encoder) writeByte(b byte) error {
	if T.bufferPos+1 > len(T.buffer) {
		if err := T.Flush(); err != nil {
			return err
		}
	}
	T.buffer[T.bufferPos] = b
	T.bufferPos++
	return nil
}

func (T *Encoder) WriteByte(b byte) error {
	if T.packetPos != T.packetLength {
		return ErrWrongNumberOfBytes
	}

	T.packetType = 0
	T.packetLength = 0
	T.packetPos = 0
	return T.writeByte(b)
}

func (T *Encoder) Next(typ Type, length int) error {
	if T.packetPos != T.packetLength {
		return ErrWrongNumberOfBytes
	}

	if typ != 0 {
		if err := T.writeByte(byte(typ)); err != nil {
			return err
		}
	}

	if T.bufferPos+4 > len(T.buffer) {
		if err := T.Flush(); err != nil {
			return err
		}
	}
	binary.BigEndian.PutUint32(T.buffer[T.bufferPos:T.bufferPos+4], uint32(length+4))
	T.bufferPos += 4

	T.packetType = typ
	T.packetLength = length
	T.packetPos = 0

	return nil
}

func (T *Encoder) Type() Type {
	return T.packetType
}

func (T *Encoder) Length() int {
	return T.packetLength
}

func (T *Encoder) Position() int {
	return T.packetPos
}

func (T *Encoder) Uint8(v uint8) error {
	err := T.writeByte(v)
	T.packetPos += 1
	return err
}

func (T *Encoder) Uint16(v uint16) error {
	if T.bufferPos+2 > len(T.buffer) {
		if err := T.Flush(); err != nil {
			return err
		}
	}
	binary.BigEndian.PutUint16(T.buffer[T.bufferPos:T.bufferPos+2], v)
	T.bufferPos += 2
	T.packetPos += 2
	return nil
}

func (T *Encoder) Uint32(v uint32) error {
	if T.bufferPos+4 > len(T.buffer) {
		if err := T.Flush(); err != nil {
			return err
		}
	}
	binary.BigEndian.PutUint32(T.buffer[T.bufferPos:T.bufferPos+4], v)
	T.bufferPos += 4
	T.packetPos += 4
	return nil
}

func (T *Encoder) Uint64(v uint64) error {
	if T.bufferPos+8 > len(T.buffer) {
		if err := T.Flush(); err != nil {
			return err
		}
	}
	binary.BigEndian.PutUint64(T.buffer[T.bufferPos:T.bufferPos+8], v)
	T.bufferPos += 8
	T.packetPos += 8
	return nil
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
	for len(v) > 0 {
		n := copy(T.buffer[T.bufferPos:], v)
		T.bufferPos += n
		T.packetPos += n
		v = v[n:]
		if T.bufferPos >= len(T.buffer) {
			if err := T.Flush(); err != nil {
				return err
			}
		}
	}
	if err := T.writeByte(0); err != nil {
		return err
	}
	T.packetPos += 1
	return nil
}
