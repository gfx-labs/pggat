package packets

import (
	"pggat2/lib/zap"
)

func ReadDescribe(in zap.Inspector) (which uint8, target string, ok bool) {
	in.Reset()
	if in.Type() != Describe {
		return
	}
	which, ok = in.Uint8()
	if !ok {
		return
	}
	target, ok = in.String()
	if !ok {
		return
	}
	return
}

func WriteDescribe(out zap.Builder, which uint8, target string) {
	out.Reset()
	out.Type(Describe)
	out.Uint8(which)
	out.String(target)
}
