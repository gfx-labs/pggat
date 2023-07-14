package packets

import (
	"pggat2/lib/zap"
)

func ReadPasswordMessage(in zap.In) (string, bool) {
	in.Reset()
	if in.Type() != AuthenticationResponse {
		return "", false
	}
	password, ok := in.String()
	if !ok {
		return "", false
	}
	return password, true
}

func WritePasswordMessage(out zap.Out, password string) {
	out.Reset()
	out.Type(AuthenticationResponse)
	out.String(password)
}
