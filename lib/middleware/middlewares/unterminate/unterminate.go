package unterminate

import (
	"io"

	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
)

type Unterminate struct {
	pnet.ReadWriter
}

func MakeUnterminate(inner pnet.ReadWriter) Unterminate {
	return Unterminate{
		ReadWriter: inner,
	}
}

func (T Unterminate) Read() (packet.In, error) {
	in, err := T.ReadWriter.Read()
	if err != nil {
		return packet.In{}, err
	}
	if in.Type() == packet.Terminate {
		return packet.In{}, io.EOF
	}
	return in, nil
}

var _ pnet.ReadWriter = Unterminate{}
