package packets

import "pggat2/lib/pnet/packet"

func ReadExecute(in packet.In) (target string, maxRows int32, ok bool) {
	in.Reset()
	if in.Type() != packet.Execute {
		return
	}
	target, ok = in.String()
	if !ok {
		return
	}
	maxRows, ok = in.Int32()
	if !ok {
		return
	}
	return
}

func WriteExecute(out packet.Out, target string, maxRows int32) {
	out.Reset()
	out.Type(packet.Execute)
	out.String(target)
	out.Int32(maxRows)
}
