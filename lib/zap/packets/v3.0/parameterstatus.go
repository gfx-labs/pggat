package packets

import "pggat2/lib/zap"

func ReadParameterStatus(in zap.Inspector) (key, value string, ok bool) {
	in.Reset()
	if in.Type() != ParameterStatus {
		return
	}
	key, ok = in.String()
	if !ok {
		return
	}
	value, ok = in.String()
	if !ok {
		return
	}
	return
}

func WriteParameterStatus(out zap.Builder, key, value string) {
	out.Reset()
	out.Type(ParameterStatus)
	out.String(key)
	out.String(value)
}
