package zap

import (
	"encoding/binary"
	"io"
	"math"

	"pggat2/lib/util/decorator"
	"pggat2/lib/util/slices"
)

type Buf struct {
	noCopy decorator.NoCopy

	pos int
	buf []byte
	rev int
}

func (T *Buf) assertRev(rev int) {
	// this check can be turned off when in production mode (for dev, this is helpful though)
	if T.rev != rev {
		panic("use after resource release")
	}
}

func (T *Buf) ReadByte(reader io.Reader) (byte, error) {
	T.rev++
	T.pos = 0

	T.buf = slices.Resize(T.buf, 1)
	_, err := io.ReadFull(reader, T.buf)
	if err != nil {
		return 0, err
	}
	return T.buf[0], nil
}

func (T *Buf) Read(reader io.Reader, typed bool) (In, error) {
	T.rev++
	T.pos = 0

	// read header
	T.buf = slices.Resize(T.buf, 5)
	var err error
	if typed {
		_, err = io.ReadFull(reader, T.buf)
	} else {
		_, err = io.ReadFull(reader, T.buf[1:])
	}
	if err != nil {
		return In{}, err
	}

	// extract length
	length := binary.BigEndian.Uint32(T.buf[1:])

	// read payload
	T.buf = slices.Resize(T.buf, int(length)+1)
	_, err = io.ReadFull(reader, T.buf[5:])
	if err != nil {
		return In{}, err
	}

	return In{
		buf: T,
		rev: T.rev,
	}, nil
}

func (T *Buf) WriteByte(writer io.Writer, b byte) error {
	T.rev++
	T.pos = 0

	T.buf = slices.Resize(T.buf, 1)
	T.buf[0] = b
	_, err := writer.Write(T.buf)
	return err
}

func (T *Buf) Write() Out {
	T.rev++
	T.pos = 0

	T.buf = slices.Resize(T.buf, 5)
	T.buf[0] = 0

	return Out{
		buf: T,
		rev: T.rev,
	}
}

func (T *Buf) full() []byte {
	// put length
	binary.BigEndian.PutUint32(T.buf[1:], uint32(len(T.buf)-1))

	if T.readType() == 0 {
		// untyped
		return T.buf[1:]
	} else {
		// typed
		return T.buf
	}
}

// read methods

func (T *Buf) resetRead() {
	T.pos = 0
}

func (T *Buf) readType() Type {
	return Type(T.buf[0])
}

func (T *Buf) remaining() []byte {
	return T.buf[T.pos+5:]
}

func (T *Buf) readUint8() (uint8, bool) {
	rem := T.remaining()
	if len(rem) < 1 {
		return 0, false
	}
	T.pos += 1
	return rem[0], true
}

func (T *Buf) readUint16() (uint16, bool) {
	rem := T.remaining()
	if len(rem) < 2 {
		return 0, false
	}
	T.pos += 2
	return binary.BigEndian.Uint16(rem), true
}

func (T *Buf) readUint32() (uint32, bool) {
	rem := T.remaining()
	if len(rem) < 4 {
		return 0, false
	}
	T.pos += 4
	return binary.BigEndian.Uint32(rem), true
}

func (T *Buf) readUint64() (uint64, bool) {
	rem := T.remaining()
	if len(rem) < 8 {
		return 0, false
	}
	T.pos += 8
	return binary.BigEndian.Uint64(rem), true
}

func (T *Buf) readInt8() (int8, bool) {
	v, ok := T.readUint8()
	if !ok {
		return 0, false
	}
	return int8(v), true
}

func (T *Buf) readInt16() (int16, bool) {
	v, ok := T.readUint16()
	if !ok {
		return 0, false
	}
	return int16(v), true
}

func (T *Buf) readInt32() (int32, bool) {
	v, ok := T.readUint32()
	if !ok {
		return 0, false
	}
	return int32(v), true
}

func (T *Buf) readInt64() (int64, bool) {
	v, ok := T.readUint64()
	if !ok {
		return 0, false
	}
	return int64(v), true
}

func (T *Buf) readFloat32() (float32, bool) {
	v, ok := T.readUint32()
	if !ok {
		return 0, false
	}
	return math.Float32frombits(v), true
}

func (T *Buf) readFloat64() (float64, bool) {
	v, ok := T.readUint64()
	if !ok {
		return 0, false
	}
	return math.Float64frombits(v), true
}

func (T *Buf) readString() (string, bool) {
	rem := T.remaining()
	for i, c := range rem {
		if c == 0 {
			T.pos += i + 1
			return string(rem[:i]), true
		}
	}
	return "", false
}

func (T *Buf) readBytes(b []byte) bool {
	rem := T.remaining()
	if len(rem) < len(b) {
		return false
	}
	T.pos += len(b)
	copy(b, rem)
	return true
}

func (T *Buf) readUnsafeBytes(count int) ([]byte, bool) {
	rem := T.remaining()
	if len(rem) < count {
		return nil, false
	}
	T.pos += count
	return rem[:count], true
}

// write methods

func (T *Buf) resetWrite() {
	T.buf = slices.Resize(T.buf, 5)
	T.buf[0] = 0
}

func (T *Buf) writeType(typ Type) {
	T.buf[0] = byte(typ)
}

func (T *Buf) writeUint8(v uint8) {
	T.buf = append(T.buf, v)
}

func (T *Buf) writeUint16(v uint16) {
	T.buf = binary.BigEndian.AppendUint16(T.buf, v)
}

func (T *Buf) writeUint32(v uint32) {
	T.buf = binary.BigEndian.AppendUint32(T.buf, v)
}

func (T *Buf) writeUint64(v uint64) {
	T.buf = binary.BigEndian.AppendUint64(T.buf, v)
}

func (T *Buf) writeInt8(v int8) {
	T.writeUint8(uint8(v))
}

func (T *Buf) writeInt16(v int16) {
	T.writeUint16(uint16(v))
}

func (T *Buf) writeInt32(v int32) {
	T.writeUint32(uint32(v))
}

func (T *Buf) writeInt64(v int64) {
	T.writeUint64(uint64(v))
}

func (T *Buf) writeFloat32(v float32) {
	T.writeUint32(math.Float32bits(v))
}

func (T *Buf) writeFloat64(v float64) {
	T.writeUint64(math.Float64bits(v))
}

func (T *Buf) writeString(v string) {
	T.buf = append(T.buf, v...)
	T.buf = append(T.buf, 0)
}

func (T *Buf) writeBytes(v []byte) {
	T.buf = append(T.buf, v...)
}
