package packets

import "pggat2/lib/pnet/packet"

func ReadAuthenticationOk(in packet.In) bool {
	in.Reset()
	if in.Type() != packet.Authentication {
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

func WriteAuthenticationOk(out packet.Out) {
	out.Reset()
	out.Type(packet.Authentication)
	out.Int32(0)
}
