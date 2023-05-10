package unterminate

import (
	"io"

	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
)

type Unterminate struct {
	pnet.ReadWriteSender
}

func MakeUnterminate(inner pnet.ReadWriteSender) Unterminate {
	return Unterminate{
		ReadWriteSender: inner,
	}
}

func (T Unterminate) Read() (packet.In, error) {
	in, err := T.ReadWriteSender.Read()
	if err != nil {
		return packet.In{}, err
	}
	if in.Type() == packet.Terminate {
		return packet.In{}, io.EOF
	}
	return in, nil
}

var _ pnet.ReadWriteSender = Unterminate{}
