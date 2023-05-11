package zap

type Out struct {
	buf *Buf
	rev int
}

func (T Out) Reset() {
	T.buf.assertRev(T.rev)
	T.buf.resetWrite()
}

func (T Out) Full() []byte {
	T.buf.assertRev(T.rev)
	return T.buf.full()
}

func (T Out) Type(typ Type) {
	T.buf.assertRev(T.rev)
	T.buf.writeType(typ)
}

func (T Out) Int8(v int8) {
	T.buf.assertRev(T.rev)
	T.buf.writeInt8(v)
}

func (T Out) Int16(v int16) {
	T.buf.assertRev(T.rev)
	T.buf.writeInt16(v)
}

func (T Out) Int32(v int32) {
	T.buf.assertRev(T.rev)
	T.buf.writeInt32(v)
}

func (T Out) Int64(v int64) {
	T.buf.assertRev(T.rev)
	T.buf.writeInt64(v)
}

func (T Out) Uint8(v uint8) {
	T.buf.assertRev(T.rev)
	T.buf.writeUint8(v)
}

func (T Out) Uint16(v uint16) {
	T.buf.assertRev(T.rev)
	T.buf.writeUint16(v)
}

func (T Out) Uint32(v uint32) {
	T.buf.assertRev(T.rev)
	T.buf.writeUint32(v)
}

func (T Out) Uint64(v uint64) {
	T.buf.assertRev(T.rev)
	T.buf.writeUint64(v)
}

func (T Out) Float32(v float32) {
	T.buf.assertRev(T.rev)
	T.buf.writeFloat32(v)
}

func (T Out) Float64(v float64) {
	T.buf.assertRev(T.rev)
	T.buf.writeFloat64(v)
}

func (T Out) String(v string) {
	T.buf.assertRev(T.rev)
	T.buf.writeString(v)
}

func (T Out) Bytes(v []byte) {
	T.buf.assertRev(T.rev)
	T.buf.writeBytes(v)
}
