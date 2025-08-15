package fed

import "fmt"

type Packet interface {
	fmt.Stringer

	Type() Type
	TypeName() string
	Length() int

	WriteTo(encoder *Encoder) error
}

type ReadablePacket interface {
	Packet

	ReadFrom(decoder *Decoder) error
}

type PendingPacket struct {
	Decoder *Decoder
}

func (T PendingPacket) Type() Type {
	return T.Decoder.Type()
}

func (T PendingPacket) TypeName() string {
	// 0 alloc
	return string(T.Type())
}

func (T PendingPacket) String() string {
	return T.TypeName()
}

func (T PendingPacket) Length() int {
	return T.Decoder.Length()
}

func (T PendingPacket) WriteTo(encoder *Encoder) error {
	for T.Decoder.Position() < T.Decoder.Length() {
		if _, err := encoder.ReadFrom(T.Decoder); err != nil {
			return err
		}
	}
	return nil
}

var _ Packet = PendingPacket{}

func ToConcrete[T any, PT interface {
	ReadFrom(decoder *Decoder) error
	*T
}](value PT, packet Packet) error {
	switch p := packet.(type) {
	case PT:
		*value = *p
		return nil
	case PendingPacket:
		return value.ReadFrom(p.Decoder)
	default:
		panic("incompatible packet types")
	}
}
