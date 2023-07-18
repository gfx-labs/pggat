package zap3

import (
	"encoding/binary"
	"io"
	"math"
	"net"

	"pggat2/lib/util/slices"
)

type Packets struct {
	packets net.Buffers
}

func (T *Packets) WriteTo(w io.Writer) (int64, error) {
	return T.packets.WriteTo(w)
}

func (T *Packets) Append(packet []byte) {
	T.packets = append(T.packets, packet)
}

func (T *Packets) Clear() {
	T.packets = T.packets[:0]
}

type PacketReader []byte

func (T *PacketReader) ReadInt8() (int8, bool) {
	if v, ok := T.ReadUint8(); ok {
		return int8(v), true
	}
	return 0, false
}

func (T *PacketReader) ReadInt16() (int16, bool) {
	if v, ok := T.ReadUint16(); ok {
		return int16(v), true
	}
	return 0, false
}

func (T *PacketReader) ReadInt32() (int32, bool) {
	if v, ok := T.ReadUint32(); ok {
		return int32(v), true
	}
	return 0, false
}

func (T *PacketReader) ReadInt64() (int64, bool) {
	if v, ok := T.ReadUint64(); ok {
		return int64(v), true
	}
	return 0, false
}

func (T *PacketReader) ReadUint8() (uint8, bool) {
	if len(*T) < 1 {
		return 0, false
	}

	v := (*T)[0]
	*T = (*T)[1:]
	return v, true
}

func (T *PacketReader) ReadUint16() (uint16, bool) {
	if len(*T) < 2 {
		return 0, false
	}

	v := binary.BigEndian.Uint16(*T)
	*T = (*T)[2:]
	return v, true
}

func (T *PacketReader) ReadUint32() (uint32, bool) {
	if len(*T) < 4 {
		return 0, false
	}

	v := binary.BigEndian.Uint32(*T)
	*T = (*T)[4:]
	return v, true
}

func (T *PacketReader) ReadUint64() (uint64, bool) {
	if len(*T) < 8 {
		return 0, false
	}

	v := binary.BigEndian.Uint64(*T)
	*T = (*T)[8:]
	return v, true
}

func (T *PacketReader) ReadFloat32() (float32, bool) {
	if v, ok := T.ReadUint32(); ok {
		return math.Float32frombits(v), true
	}

	return 0, false
}

func (T *PacketReader) ReadFloat64() (float64, bool) {
	if v, ok := T.ReadUint64(); ok {
		return math.Float64frombits(v), true
	}

	return 0, false
}

func (T *PacketReader) ReadString() (string, bool) {
	for i, b := range *T {
		if b == 0 {
			v := (*T)[:i]
			*T = (*T)[i+1:]
			return string(v), true
		}
	}

	return "", false
}

func (T *PacketReader) ReadBytes(b []byte) bool {
	if len(*T) < len(b) {
		return false
	}

	copy(b, *T)
	*T = (*T)[len(b):]
	return true
}

type PacketWriter []byte

func (T *PacketWriter) WriteInt8(v int8) {
	T.WriteUint8(uint8(v))
}

func (T *PacketWriter) WriteInt16(v int16) {
	T.WriteUint16(uint16(v))
}

func (T *PacketWriter) WriteInt32(v int32) {
	T.WriteUint32(uint32(v))
}

func (T *PacketWriter) WriteInt64(v int64) {
	T.WriteUint64(uint64(v))
}

func (T *PacketWriter) WriteUint8(v uint8) {
	*T = append(*T, v)
}

func (T *PacketWriter) WriteUint16(v uint16) {
	*T = binary.BigEndian.AppendUint16(*T, v)
}

func (T *PacketWriter) WriteUint32(v uint32) {
	*T = binary.BigEndian.AppendUint32(*T, v)
}

func (T *PacketWriter) WriteUint64(v uint64) {
	*T = binary.BigEndian.AppendUint64(*T, v)
}

func (T *PacketWriter) WriteFloat32(v float32) {
	T.WriteUint32(math.Float32bits(v))
}

func (T *PacketWriter) WriteFloat64(v float64) {
	T.WriteUint64(math.Float64bits(v))
}

func (T *PacketWriter) WriteString(v string) {
	*T = append(*T, v...)
	T.WriteUint8(0)
}

func (T *PacketWriter) WriteBytes(v []byte) {
	*T = append(*T, v...)
}

type ReadablePacket struct {
	PacketReader
	typ Type
}

func (T *ReadablePacket) ReadType() Type {
	return T.typ
}

type Packet struct {
	PacketWriter
}

func (T *Packet) ReadFrom(r io.Reader) (int64, error) {
	T.PacketWriter = slices.Resize(T.PacketWriter, 5)
	n, err := io.ReadFull(r, T.PacketWriter)
	if err != nil {
		return int64(n), err
	}
	length := T.Length()
	T.PacketWriter = slices.Resize(T.PacketWriter, int(length)+1)
	m, err := io.ReadFull(r, T.Payload())
	return int64(n + m), err
}

func (T *Packet) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(T.PacketWriter)
	return int64(n), err
}

func (T *Packet) Length() uint32 {
	return binary.BigEndian.Uint32(T.PacketWriter[1:])
}

func (T *Packet) Payload() []byte {
	return T.PacketWriter[5:]
}

func (T *Packet) WriteType(v Type) {
	T.PacketWriter[0] = v
}

func (T *Packet) Read() ReadablePacket {
	return ReadablePacket{
		typ:          T.PacketWriter[0],
		PacketReader: T.Payload(),
	}
}

type UntypedReadablePacket struct {
	PacketReader
}

type UntypedPacket struct {
	PacketWriter
}

func (T *UntypedPacket) ReadFrom(r io.Reader) (int64, error) {
	T.PacketWriter = slices.Resize(T.PacketWriter, 4)
	n, err := io.ReadFull(r, T.PacketWriter)
	if err != nil {
		return int64(n), err
	}
	length := T.Length()
	T.PacketWriter = slices.Resize(T.PacketWriter, int(length))
	m, err := io.ReadFull(r, T.Payload())
	return int64(n + m), err
}

func (T *UntypedPacket) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(T.PacketWriter)
	return int64(n), err
}

func (T *UntypedPacket) Length() uint32 {
	return binary.BigEndian.Uint32(T.PacketWriter)
}

func (T *UntypedPacket) Payload() []byte {
	return T.PacketWriter[4:]
}

func (T *UntypedPacket) Read() UntypedReadablePacket {
	return UntypedReadablePacket{
		PacketReader: PacketReader(T.Payload()),
	}
}
