package eqp

import (
	"pggat2/lib/global"
	"pggat2/lib/util/slices"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type PreparedStatement struct {
	raw []byte
}

func ReadParse(in zap.In) (destination string, preparedStatement PreparedStatement, ok bool) {
	in.Reset()
	if in.Type() != packets.Parse {
		return
	}
	destination, ok = in.String()
	if !ok {
		return
	}
	full := zap.InToOut(in).Full()
	preparedStatement.raw = global.GetBytes(int32(len(full)))
	copy(preparedStatement.raw, full)
	return
}

func (T *PreparedStatement) Done() {
	global.PutBytes(T.raw)
	T.raw = nil
}

func (T *PreparedStatement) Equal(rhs *PreparedStatement) bool {
	return slices.Equal(T.raw, rhs.raw)
}

func (T *PreparedStatement) Clone() PreparedStatement {
	raw := global.GetBytes(int32(len(T.raw)))
	copy(raw, T.raw)
	return PreparedStatement{
		raw: raw,
	}
}
