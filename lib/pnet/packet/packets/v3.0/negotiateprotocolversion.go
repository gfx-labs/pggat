package packets

import "pggat2/lib/pnet/packet"

func ReadNegotiateProtocolVersion(in packet.In) (minorProtocolVersion int32, unrecognizedOptions []string, ok bool) {
	in.Reset()
	if in.Type() != packet.NegotiateProtocolVersion {
		return
	}
	minorProtocolVersion, ok = in.Int32()
	if !ok {
		return
	}
	var numUnrecognizedOptions int32
	numUnrecognizedOptions, ok = in.Int32()
	if !ok {
		return
	}
	unrecognizedOptions = make([]string, 0, numUnrecognizedOptions)
	for i := 0; i < int(numUnrecognizedOptions); i++ {
		var unrecognizedOption string
		unrecognizedOption, ok = in.String()
		if !ok {
			return
		}
		unrecognizedOptions = append(unrecognizedOptions, unrecognizedOption)
	}
	ok = true
	return
}

func WriteNegotiateProtocolVersion(out packet.Out, minorProtocolVersion int32, unrecognizedOptions []string) {
	out.Reset()
	out.Type(packet.NegotiateProtocolVersion)
	out.Int32(minorProtocolVersion)
	out.Int32(int32(len(unrecognizedOptions)))
	for _, option := range unrecognizedOptions {
		out.String(option)
	}
}
