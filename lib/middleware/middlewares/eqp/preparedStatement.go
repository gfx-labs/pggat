package eqp

import (
	"hash/maphash"

	"pggat2/lib/global"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type PreparedStatement struct {
	raw  []byte
	hash uint64
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
	preparedStatement.hash = maphash.Bytes(seed, full)
	preparedStatement.raw = global.GetBytes(int32(len(full)))
	copy(preparedStatement.raw, full)
	return
}

func (T *PreparedStatement) Done() {
	global.PutBytes(T.raw)
	T.raw = nil
}
