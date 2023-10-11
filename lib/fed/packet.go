package fed

import "io"

type Packet interface {
	Type() Type
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

func (T PendingPacket) Length() int {
	return T.Decoder.Length()
}

func (T PendingPacket) WriteTo(encoder *Encoder) error {
	count := T.Decoder.len - T.Decoder.pos
	limited := io.LimitedReader{
		R: &T.Decoder.Reader,
		N: int64(count),
	}
	for limited.N > 0 {
		if _, err := encoder.Writer.ReadFrom(&limited); err != nil {
			return err
		}
	}
	T.Decoder.pos += count
	encoder.pos += count
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
