package packets

import (
	"pggat2/lib/zap"
)

func ReadAuthenticationSASLContinue(in zap.Inspector) ([]byte, bool) {
	in.Reset()
	if in.Type() != Authentication {
		return nil, false
	}
	method, ok := in.Int32()
	if !ok {
		return nil, false
	}
	if method != 11 {
		return nil, false
	}
	return in.Remaining(), true
}

func WriteAuthenticationSASLContinue(out zap.Builder, resp []byte) {
	out.Reset()
	out.Type(Authentication)
	out.Int32(11)
	out.Bytes(resp)
}
