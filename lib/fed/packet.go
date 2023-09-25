package fed

import (
	"encoding/binary"
	"math"

	"pggat/lib/util/slices"
)

type Packet []byte

func NewPacket(typ Type, size ...int) Packet {
	return Packet(nil).Reset(typ, size...)
}

func (T Packet) Reset(typ Type, size ...int) Packet {
	packet := T
	c := 5
	if len(size) > 0 {
		c += size[0]
	}

	if cap(packet) < c {
		packet = make([]byte, 5, c)
	} else {
		packet = slices.Resize(packet, 5)
	}
	packet[0] = byte(typ)
	packet[1] = 0
	packet[2] = 0
	packet[3] = 0
	packet[4] = 0
	return packet
}

func (T Packet) Payload() PacketFragment {
	return PacketFragment(T[5:])
}

func (T Packet) Bytes() []byte {
	binary.BigEndian.PutUint32(T[1:], uint32(len(T)-1))

	if T.Type() == 0 {
		return T[1:]
	}
	return T
}

func (T Packet) Type() Type {
	return Type(T[0])
}

func (T Packet) AppendUint8(v uint8) Packet {
	return append(T, v)
}

func (T Packet) AppendUint16(v uint16) Packet {
	return binary.BigEndian.AppendUint16(T, v)
}

func (T Packet) AppendUint32(v uint32) Packet {
	return binary.BigEndian.AppendUint32(T, v)
}

func (T Packet) AppendUint64(v uint64) Packet {
	return binary.BigEndian.AppendUint64(T, v)
}

func (T Packet) AppendInt8(v int8) Packet {
	return T.AppendUint8(uint8(v))
}

func (T Packet) AppendInt16(v int16) Packet {
	return T.AppendUint16(uint16(v))
}

func (T Packet) AppendInt32(v int32) Packet {
	return T.AppendUint32(uint32(v))
}

func (T Packet) AppendInt64(v int64) Packet {
	return T.AppendUint64(uint64(v))
}

func (T Packet) AppendFloat32(v float32) Packet {
	return T.AppendUint32(math.Float32bits(v))
}

func (T Packet) AppendFloat64(v float64) Packet {
	return T.AppendUint64(math.Float64bits(v))
}

func (T Packet) AppendString(v string) Packet {
	return append(append(T, v...), 0)
}

func (T Packet) AppendBytes(v []byte) Packet {
	return append(T, v...)
}

func (T Packet) ReadUint8(v *uint8) PacketFragment {
	return T.Payload().ReadUint8(v)
}

func (T Packet) ReadUint16(v *uint16) PacketFragment {
	return T.Payload().ReadUint16(v)
}

func (T Packet) ReadUint32(v *uint32) PacketFragment {
	return T.Payload().ReadUint32(v)
}

func (T Packet) ReadUint64(v *uint64) PacketFragment {
	return T.Payload().ReadUint64(v)
}

func (T Packet) ReadInt8(v *int8) PacketFragment {
	return T.Payload().ReadInt8(v)
}

func (T Packet) ReadInt16(v *int16) PacketFragment {
	return T.Payload().ReadInt16(v)
}

func (T Packet) ReadInt32(v *int32) PacketFragment {
	return T.Payload().ReadInt32(v)
}

func (T Packet) ReadInt64(v *int64) PacketFragment {
	return T.Payload().ReadInt64(v)
}

func (T Packet) ReadFloat32(v *float32) PacketFragment {
	return T.Payload().ReadFloat32(v)
}

func (T Packet) ReadFloat64(v *float64) PacketFragment {
	return T.Payload().ReadFloat64(v)
}

func (T Packet) ReadString(v *string) PacketFragment {
	return T.Payload().ReadString(v)
}

func (T Packet) ReadBytes(v []byte) PacketFragment {
	return T.Payload().ReadBytes(v)
}

type PacketFragment []byte

func (T PacketFragment) ReadUint8(v *uint8) PacketFragment {
	if len(T) < 1 {
		return T
	}

	*v = T[0]
	return T[1:]
}

func (T PacketFragment) ReadUint16(v *uint16) PacketFragment {
	if len(T) < 2 {
		return T
	}

	*v = binary.BigEndian.Uint16(T)
	return T[2:]
}

func (T PacketFragment) ReadUint32(v *uint32) PacketFragment {
	if len(T) < 4 {
		return T
	}

	*v = binary.BigEndian.Uint32(T)
	return T[4:]
}

func (T PacketFragment) ReadUint64(v *uint64) PacketFragment {
	if len(T) < 8 {
		return T
	}

	*v = binary.BigEndian.Uint64(T)
	return T[8:]
}

func (T PacketFragment) ReadInt8(v *int8) PacketFragment {
	var vv uint8
	n := T.ReadUint8(&vv)
	*v = int8(vv)
	return n
}

func (T PacketFragment) ReadInt16(v *int16) PacketFragment {
	var vv uint16
	n := T.ReadUint16(&vv)
	*v = int16(vv)
	return n
}

func (T PacketFragment) ReadInt32(v *int32) PacketFragment {
	var vv uint32
	n := T.ReadUint32(&vv)
	*v = int32(vv)
	return n
}

func (T PacketFragment) ReadInt64(v *int64) PacketFragment {
	var vv uint64
	n := T.ReadUint64(&vv)
	*v = int64(vv)
	return n
}

func (T PacketFragment) ReadFloat32(v *float32) PacketFragment {
	var vv uint32
	n := T.ReadUint32(&vv)
	*v = math.Float32frombits(vv)
	return n
}

func (T PacketFragment) ReadFloat64(v *float64) PacketFragment {
	var vv uint64
	n := T.ReadUint64(&vv)
	*v = math.Float64frombits(vv)
	return n
}

func (T PacketFragment) ReadString(v *string) PacketFragment {
	for i, b := range T {
		if b != '\x00' {
			continue
		}
		*v = string(T[:i])
		return T[i+1:]
	}

	return T
}

func (T PacketFragment) ReadBytes(v []byte) PacketFragment {
	if len(T) < len(v) {
		return T
	}
	copy(v, T)
	return T[len(v):]
}
