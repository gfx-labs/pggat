package eqp

import (
	"hash/maphash"

	"pggat2/lib/global"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Portal struct {
	source string
	raw    []byte
	hash   uint64
}

func ReadBind(in zap.Inspector) (destination string, portal Portal, ok bool) {
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
	full := in.Payload()
	portal.hash = maphash.Bytes(seed, full)
	portal.raw = global.GetBytes(int32(len(full)))
	copy(portal.raw, full)
	return
}

func (T *Portal) Done() {
	global.PutBytes(T.raw)
	T.raw = nil
}
