package packets

import "pggat2/lib/pnet/packet"

func ReadAuthenticationResponse(in packet.In) ([]byte, bool) {
	in.Reset()
	if in.Type() != packet.AuthenticationResponse {
		return nil, false
	}
	return in.Full(), true
}

func WriteAuthenticationResponse(out packet.Out, resp []byte) {
	out.Reset()
	out.Type(packet.AuthenticationResponse)
	out.Bytes(resp)
}
