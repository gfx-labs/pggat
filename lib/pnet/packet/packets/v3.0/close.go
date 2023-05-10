package packets

import "pggat2/lib/pnet/packet"

func ReadClose(in packet.In) (which uint8, target string, ok bool) {
	in.Reset()
	if in.Type() != packet.Close {
		return
	}
	which, ok = in.Uint8()
	if !ok {
		return
	}
	target, ok = in.String()
	if !ok {
		return
	}
	return
}

func WriteClose(out packet.Out, which uint8, target string) {
	out.Reset()
	out.Type(packet.Close)
	out.Uint8(which)
	out.String(target)
}
