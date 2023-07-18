package packets

import "pggat2/lib/zap"

func ReadAuthenticationResponse(in zap.Inspector) ([]byte, bool) {
	in.Reset()
	if in.Type() != AuthenticationResponse {
		return nil, false
	}
	return in.Remaining(), true
}

func WriteAuthenticationResponse(out zap.Builder, resp []byte) {
	out.Reset()
	out.Type(AuthenticationResponse)
	out.Bytes(resp)
}
