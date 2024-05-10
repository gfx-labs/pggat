package fed

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"strings"

	"gfx.cafe/gfx/pggat/lib/util/decorator"
)

type Decoder struct {
	noCopy decorator.NoCopy

	buffer      [defaultBufferSize]byte
	bufferWrite int
	bufferRead  int
	reader      io.Reader

	packetType   Type
	packetLength int
	packetPos    int
	decodeBuf    [8]byte
}

func NewDecoder(r io.Reader) *Decoder {
	d := new(Decoder)
	d.Reset(r)
	return d
}

func (T *Decoder) Reset(r io.Reader) {
	T.packetLength = 0
	T.packetPos = 0
	T.bufferRead = 0
	T.bufferWrite = 0
	T.reader = r
}

func (T *Decoder) refill() error {
	n, err := T.reader.Read(T.buffer[T.bufferWrite:])
	T.bufferWrite += n
	return err
}

func (T *Decoder) discard(n int) error {
	for n > 0 {
		if T.bufferWrite != 0 {
			count := min(n, T.bufferWrite-T.bufferRead)
			T.bufferRead += count
			n -= count
			if T.bufferRead == T.bufferWrite {
				T.bufferRead = 0
				T.bufferWrite = 0
			} else {
				break
			}
		}

		if err := T.refill(); err != nil {
			return err
		}
	}

	return nil
}

func (T *Decoder) read(b []byte) (n int, err error) {
	if len(b) == 0 {
		return
	}
	if T.bufferWrite != 0 {
		n = copy(b, T.buffer[T.bufferRead:T.bufferWrite])
		T.bufferRead += n
		if T.bufferRead == T.bufferWrite {
			T.bufferRead = 0
			T.bufferWrite = 0
		}
		return
	}

	if len(b) > len(T.buffer) {
		return T.reader.Read(b)
	}

	// read into buffer first
	err = T.refill()
	n = copy(b, T.buffer[T.bufferRead:T.bufferWrite])
	T.bufferRead += n
	if T.bufferRead == T.bufferWrite {
		T.bufferRead = 0
		T.bufferWrite = 0
	}
	return
}

func (T *Decoder) readFull(b []byte) (n int, err error) {
	for n < len(b) {
		var count int
		count, err = T.read(b[n:])
		n += count
		if err != nil {
			if err == io.EOF && n != 0 {
				err = io.ErrUnexpectedEOF
			}
			return
		}
	}
	return
}

func (T *Decoder) Read(b []byte) (n int, err error) {
	rem := T.packetLength - T.packetPos
	if rem == 0 {
		err = io.EOF
		return
	}
	if len(b) > rem {
		n, err = T.read(b[:rem])
	} else {
		n, err = T.read(b)
	}
	T.packetPos += n
	return
}

func (T *Decoder) Buffered() int {
	return T.bufferWrite - T.bufferRead
}

var ErrOverranReadBuffer = errors.New("overran read buffer")

func (T *Decoder) ReadByte() (byte, error) {
	rem := T.packetLength - T.packetPos
	if rem < 0 {
		return 0, ErrOverranReadBuffer
	} else if rem > 0 {
		if err := T.discard(rem); err != nil {
			return 0, err
		}
	}

	T.packetType = 0
	T.packetLength = 0
	T.packetPos = 0
	if _, err := T.readFull(T.decodeBuf[:1]); err != nil {
		return 0, err
	}
	return T.decodeBuf[0], nil
}

func (T *Decoder) Next(typed bool) error {
	rem := T.packetLength - T.packetPos
	if rem < 0 {
		return ErrOverranReadBuffer
	} else if rem > 0 {
		if err := T.discard(rem); err != nil {
			return err
		}
	}

	var err error
	if typed {
		_, err = T.readFull(T.decodeBuf[:5])
	} else {
		T.decodeBuf[0] = 0
		_, err = T.readFull(T.decodeBuf[1:5])
	}
	if err != nil {
		return err
	}
	T.packetType = Type(T.decodeBuf[0])
	T.packetLength = int(binary.BigEndian.Uint32(T.decodeBuf[1:5])) - 4
	T.packetPos = 0
	return nil
}

func (T *Decoder) Type() Type {
	return T.packetType
}

func (T *Decoder) Length() int {
	return T.packetLength
}

func (T *Decoder) Position() int {
	return T.packetPos
}

func (T *Decoder) Uint8() (uint8, error) {
	_, err := T.readFull(T.decodeBuf[:1])
	T.packetPos += 1
	return T.decodeBuf[0], err
}

func (T *Decoder) Uint16() (uint16, error) {
	_, err := T.readFull(T.decodeBuf[:2])
	T.packetPos += 2
	return binary.BigEndian.Uint16(T.decodeBuf[:2]), err
}

func (T *Decoder) Uint32() (uint32, error) {
	_, err := T.readFull(T.decodeBuf[:4])
	T.packetPos += 4
	return binary.BigEndian.Uint32(T.decodeBuf[:4]), err
}

func (T *Decoder) Uint64() (uint64, error) {
	_, err := T.readFull(T.decodeBuf[:8])
	T.packetPos += 8
	return binary.BigEndian.Uint64(T.decodeBuf[:8]), err
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
	if T.bufferWrite == 0 {
		if err := T.refill(); err != nil {
			return "", err
		}
	}

	for i, v := range T.buffer[T.bufferRead:T.bufferWrite] {
		if v == 0 {
			res := string(T.buffer[T.bufferRead : T.bufferRead+i])
			T.bufferRead += i + 1
			if T.bufferRead == T.bufferWrite {
				T.bufferRead = 0
				T.bufferWrite = 0
			}
			T.packetPos += i + 1
			return res, nil
		}
	}

	var builder strings.Builder
	builder.Write(T.buffer[T.bufferRead:T.bufferWrite])
	T.bufferRead = 0
	T.bufferWrite = 0
	for {
		if err := T.refill(); err != nil {
			T.packetPos += builder.Len()
			return builder.String(), err
		}

		for i, v := range T.buffer[T.bufferRead:T.bufferWrite] {
			if v == 0 {
				builder.Write(T.buffer[T.bufferRead : T.bufferRead+i])
				T.bufferRead += i + 1
				if T.bufferRead == T.bufferWrite {
					T.bufferRead = 0
					T.bufferWrite = 0
				}
				T.packetPos += builder.Len() + 1
				return builder.String(), nil
			}
		}

		builder.Write(T.buffer[T.bufferRead:T.bufferWrite])
		T.bufferRead = 0
		T.bufferWrite = 0
	}
}
