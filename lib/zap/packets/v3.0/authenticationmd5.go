package packets

import (
	"pggat2/lib/zap"
)

func ReadAuthenticationMD5(in zap.In) ([4]byte, bool) {
	in.Reset()
	if in.Type() != Authentication {
		return [4]byte{}, false
	}
	method, ok := in.Int32()
	if !ok {
		return [4]byte{}, false
	}
	if method != 5 {
		return [4]byte{}, false
	}
	var salt [4]byte
	ok = in.Bytes(salt[:])
	if !ok {
		return salt, false
	}
	return salt, true
}

func WriteAuthenticationMD5(out zap.Out, salt [4]byte) {
	out.Reset()
	out.Type(Authentication)
	out.Uint32(5)
	out.Bytes(salt[:])
}
