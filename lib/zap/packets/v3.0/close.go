package packets

import (
	"pggat2/lib/zap"
)

func ReadClose(in zap.In) (which uint8, target string, ok bool) {
	in.Reset()
	if in.Type() != Close {
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

func WriteClose(out zap.Out, which uint8, target string) {
	out.Reset()
	out.Type(Close)
	out.Uint8(which)
	out.String(target)
}
