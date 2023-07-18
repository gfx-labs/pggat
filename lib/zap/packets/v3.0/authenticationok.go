package packets

import (
	"pggat2/lib/zap"
)

func ReadAuthenticationOk(in zap.Inspector) bool {
	in.Reset()
	if in.Type() != Authentication {
		return false
	}
	method, ok := in.Int32()
	if !ok {
		return false
	}
	if method != 0 {
		return false
	}
	return true
}

func WriteAuthenticationOk(out zap.Builder) {
	out.Reset()
	out.Type(Authentication)
	out.Int32(0)
}
