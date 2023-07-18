package packets

import "pggat2/lib/zap"

func ReadAuthenticationSASLFinal(in zap.Inspector) ([]byte, bool) {
	in.Reset()
	if in.Type() != Authentication {
		return nil, false
	}
	method, ok := in.Int32()
	if !ok {
		return nil, false
	}
	if method != 12 {
		return nil, false
	}
	return in.Remaining(), true
}

func WriteAuthenticationSASLFinal(out zap.Builder, resp []byte) {
	out.Reset()
	out.Type(Authentication)
	out.Int32(12)
	out.Bytes(resp)
}
