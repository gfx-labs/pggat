package zap

type In struct {
	buf *Buf
	rev int
}

func (T In) Reset() {
	T.buf.assertRev(T.rev)
	T.buf.resetRead()
}

func (T In) Remaining() []byte {
	T.buf.assertRev(T.rev)
	return T.buf.remaining()
}

func (T In) Type() Type {
	T.buf.assertRev(T.rev)
	return T.buf.readType()
}

func (T In) Int8() (int8, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readInt8()
}

func (T In) Int16() (int16, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readInt16()
}

func (T In) Int32() (int32, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readInt32()
}

func (T In) Int64() (int64, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readInt64()
}

func (T In) Uint8() (uint8, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readUint8()
}

func (T In) Uint16() (uint16, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readUint16()
}

func (T In) Uint32() (uint32, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readUint32()
}

func (T In) Uint64() (uint64, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readUint64()
}

func (T In) Float32() (float32, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readFloat32()
}

func (T In) Float64() (float64, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readFloat64()
}

func (T In) StringBytes(b []byte) ([]byte, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readStringBytes(b)
}

func (T In) String() (string, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readString()
}

func (T In) Bytes(b []byte) bool {
	T.buf.assertRev(T.rev)
	return T.buf.readBytes(b)
}

func (T In) UnsafeBytes(count int) ([]byte, bool) {
	T.buf.assertRev(T.rev)
	return T.buf.readUnsafeBytes(count)
}
