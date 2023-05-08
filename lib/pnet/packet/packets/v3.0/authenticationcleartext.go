package packets

import "pggat2/lib/pnet/packet"

func ReadAuthenticationCleartext(in packet.In) bool {
	in.Reset()
	if in.Type() != packet.Authentication {
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

func WriteAuthenticationCleartext(out packet.Out) {
	out.Reset()
	out.Type(packet.Authentication)
	out.Int32(3)
}
