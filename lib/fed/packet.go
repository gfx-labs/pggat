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

func ToConcrete[T ReadablePacket](packet Packet) (T, error) {
	switch p := packet.(type) {
	case T:
		return p, nil
	case PendingPacket:
		var res T
		err := res.ReadFrom(p.Decoder)
		return res, err
	default:
		panic("incompatible packet types")
	}
}
