package packets

import "pggat2/lib/pnet/packet"

func ReadParameterStatus(in packet.In) (key, value string, ok bool) {
	in.Reset()
	if in.Type() != packet.ParameterStatus {
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

func WriteParameterStatus(out packet.Out, key, value string) {
	out.Reset()
	out.Type(packet.ParameterStatus)
	out.String(key)
	out.String(value)
}
