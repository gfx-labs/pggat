package packets

import "pggat2/lib/pnet/packet"

func ReadPasswordMessage(in packet.In) (string, bool) {
	in.Reset()
	if in.Type() != packet.AuthenticationResponse {
		return "", false
	}
	password, ok := in.String()
	if !ok {
		return "", false
	}
	return password, true
}

func WritePasswordMessage(out packet.Out, password string) {
	out.Reset()
	out.Type(packet.AuthenticationResponse)
	out.String(password)
}
