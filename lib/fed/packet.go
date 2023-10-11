package fed

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
	// TODO(garet) this should be better
	b, err := T.Decoder.Remaining()
	if err != nil {
		return err
	}
	return encoder.Bytes(b)
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
