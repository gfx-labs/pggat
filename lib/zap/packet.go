package zap

import (
	"encoding/binary"
	"io"
	"math"
	"net"

	"pggat2/lib/util/slices"
)

type PacketsEntry struct {
	typed bool
	index int
}

type Packets struct {
	packets        []*Packet
	untypedPackets []*UntypedPacket
	order          []PacketsEntry
}

func NewPackets() *Packets {
	return &Packets{}
}

func (T *Packets) WriteTo(w io.Writer) (int64, error) {
	buffers := make(net.Buffers, 0, len(T.order))

	for _, order := range T.order {
		if order.typed {
			buffers = append(buffers, T.packets[order.index].Full())
		} else {
			buffers = append(buffers, T.untypedPackets[order.index].Full())
		}
	}

	return buffers.WriteTo(w)
}

func (T *Packets) InsertBefore(i int, packet *Packet) {
	index := len(T.packets)
	T.packets = append(T.packets, packet)
	T.order = append(T.order, PacketsEntry{})
	copy(T.order[i+1:], T.order[i:])
	T.order[i] = PacketsEntry{
		typed: true,
		index: index,
	}
}

func (T *Packets) InsertUntypedBefore(i int, packet *UntypedPacket) {
	index := len(T.untypedPackets)
	T.untypedPackets = append(T.untypedPackets, packet)
	T.order = append(T.order, PacketsEntry{})
	copy(T.order[i+1:], T.order[i:])
	T.order[i] = PacketsEntry{
		typed: false,
		index: index,
	}
}

func (T *Packets) Append(packet *Packet) {
	index := len(T.packets)
	T.packets = append(T.packets, packet)
	T.order = append(T.order, PacketsEntry{
		typed: true,
		index: index,
	})
}

func (T *Packets) AppendUntyped(packet *UntypedPacket) {
	index := len(T.untypedPackets)
	T.untypedPackets = append(T.untypedPackets, packet)
	T.order = append(T.order, PacketsEntry{
		typed: true,
		index: index,
	})
}

func (T *Packets) Size() int {
	return len(T.order)
}

func (T *Packets) IsTyped(i int) bool {
	return T.order[i].typed
}

func (T *Packets) Get(i int) *Packet {
	order := T.order[i]
	if !order.typed {
		panic("Get() for untyped packet (use GetUntyped() instead)")
	}

	return T.packets[order.index]
}

func (T *Packets) GetUntyped(i int) *UntypedPacket {
	order := T.order[i]
	if order.typed {
		panic("GetUntyped() for typed packet (use Get() instead)")
	}

	return T.untypedPackets[order.index]
}

func (T *Packets) Remove(i int) {
	copy(T.order[i:], T.order[i+1:])
	T.order = T.order[:len(T.order)-1]
}

func (T *Packets) Clear() {
	T.order = T.order[:0]
	T.packets = T.packets[:0]
	T.untypedPackets = T.untypedPackets[:0]
}

func (T *Packets) Done() {
	// TODO(garet)
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

func (T *PacketReader) ReadUnsafeBytes(n int) ([]byte, bool) {
	if len(*T) < n {
		return nil, false
	}
	v := (*T)[:n]
	*T = (*T)[n:]
	return v, true
}

func (T *PacketReader) ReadUnsafeRemaining() []byte {
	v := *T
	*T = nil
	return v
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

func NewPacket() *Packet {
	return &Packet{
		PacketWriter{
			0, 0, 0, 0, 4,
		},
	}
}

func (T *Packet) ReadFrom(r io.Reader) (int64, error) {
	T.PacketWriter = slices.Resize(T.PacketWriter, 5)
	n, err := io.ReadFull(r, T.PacketWriter)
	if err != nil {
		return int64(n), err
	}
	length := binary.BigEndian.Uint32(T.PacketWriter[1:])
	T.PacketWriter = slices.Resize(T.PacketWriter, int(length)+1)
	m, err := io.ReadFull(r, T.Payload())
	return int64(n + m), err
}

func (T *Packet) updateLength() {
	binary.BigEndian.PutUint32(T.PacketWriter[1:], T.Length())
}

func (T *Packet) Full() []byte {
	T.updateLength()
	return T.PacketWriter
}

func (T *Packet) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(T.Full())
	return int64(n), err
}

func (T *Packet) Length() uint32 {
	return uint32(len(T.PacketWriter)) - 1
}

func (T *Packet) Payload() []byte {
	return T.PacketWriter[5:]
}

func (T *Packet) WriteType(v Type) {
	T.PacketWriter[0] = byte(v)
	T.PacketWriter = T.PacketWriter[:5]
}

func (T *Packet) ReadType() Type {
	return Type(T.PacketWriter[0])
}

func (T *Packet) Read() ReadablePacket {
	return ReadablePacket{
		typ:          T.ReadType(),
		PacketReader: T.Payload(),
	}
}

func (T *Packet) Done() {
	// TODO(garet)
}

type UntypedReadablePacket struct {
	PacketReader
}

type UntypedPacket struct {
	PacketWriter
}

func NewUntypedPacket() *UntypedPacket {
	return &UntypedPacket{
		PacketWriter{
			0, 0, 0, 4,
		},
	}
}

func (T *UntypedPacket) ReadFrom(r io.Reader) (int64, error) {
	T.PacketWriter = slices.Resize(T.PacketWriter, 4)
	n, err := io.ReadFull(r, T.PacketWriter)
	if err != nil {
		return int64(n), err
	}
	length := binary.BigEndian.Uint32(T.PacketWriter)
	T.PacketWriter = slices.Resize(T.PacketWriter, int(length))
	m, err := io.ReadFull(r, T.Payload())
	return int64(n + m), err
}

func (T *UntypedPacket) updateLength() {
	binary.BigEndian.PutUint32(T.PacketWriter, T.Length())
}

func (T *UntypedPacket) Full() []byte {
	T.updateLength()
	return T.PacketWriter
}

func (T *UntypedPacket) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(T.Full())
	return int64(n), err
}

func (T *UntypedPacket) Length() uint32 {
	return uint32(len(T.PacketWriter))
}

func (T *UntypedPacket) Payload() []byte {
	return T.PacketWriter[4:]
}

func (T *UntypedPacket) Read() UntypedReadablePacket {
	return UntypedReadablePacket{
		PacketReader: PacketReader(T.Payload()),
	}
}

func (T *UntypedPacket) Done() {
	// TODO(garet)
}
