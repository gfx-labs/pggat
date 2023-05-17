package eqp

import (
	"pggat2/lib/global"
	"pggat2/lib/util/slices"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Portal struct {
	source string
	raw    []byte
}

func ReadBind(in zap.In) (destination string, portal Portal, ok bool) {
	in.Reset()
	if in.Type() != packets.Bind {
		return
	}
	destination, ok = in.String()
	if !ok {
		return
	}
	portal.source, ok = in.String()
	if !ok {
		return
	}
	full := zap.InToOut(in).Full()
	portal.raw = global.GetBytes(int32(len(full)))
	copy(portal.raw, full)
	return
}

func (T *Portal) Done() {
	global.PutBytes(T.raw)
	T.raw = nil
}

func (T *Portal) Equal(rhs *Portal) bool {
	return slices.Equal(T.raw, rhs.raw)
}

func (T *Portal) Clone() Portal {
	raw := global.GetBytes(int32(len(T.raw)))
	copy(raw, T.raw)
	return Portal{
		source: T.source,
		raw:    raw,
	}
}
