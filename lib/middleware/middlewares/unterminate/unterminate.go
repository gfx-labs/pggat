package unterminate

import (
	"io"

	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Unterminate struct {
	zap.ReadWriter
}

func MakeUnterminate(inner zap.ReadWriter) Unterminate {
	return Unterminate{
		ReadWriter: inner,
	}
}

func (T Unterminate) Read() (zap.In, error) {
	in, err := T.ReadWriter.Read()
	if err != nil {
		return zap.In{}, err
	}
	if in.Type() == packets.Terminate {
		return zap.In{}, io.EOF
	}
	return in, nil
}

var _ zap.ReadWriter = Unterminate{}
