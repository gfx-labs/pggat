package packets

import (
	"pggat2/lib/zap"
)

func ReadAuthenticationCleartext(in zap.In) bool {
	in.Reset()
	if in.Type() != Authentication {
		return false
	}
	method, ok := in.Int32()
	if !ok {
		return false
	}
	if method != 3 {
		return false
	}
	return true
}

func WriteAuthenticationCleartext(out zap.Out) {
	out.Reset()
	out.Type(Authentication)
	out.Int32(3)
}
