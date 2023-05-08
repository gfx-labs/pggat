package packets

import "pggat2/lib/pnet/packet"

func ReadAuthenticationSASLContinue(in packet.In) ([]byte, bool) {
	in.Reset()
	if in.Type() != packet.Authentication {
		return nil, false
	}
	method, ok := in.Int32()
	if !ok {
		return nil, false
	}
	if method != 11 {
		return nil, false
	}
	return in.Full(), true
}

func WriteAuthenticationSASLContinue(out packet.Out, resp []byte) {
	out.Reset()
	out.Type(packet.Authentication)
	out.Int32(11)
	out.Bytes(resp)
}
