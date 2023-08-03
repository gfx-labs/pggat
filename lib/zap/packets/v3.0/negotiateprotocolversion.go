package packets

import "pggat2/lib/zap"

func ReadNegotiateProtocolVersion(in zap.ReadablePacket) (minorProtocolVersion int32, unrecognizedOptions []string, ok bool) {
	if in.ReadType() != NegotiateProtocolVersion {
		return
	}
	minorProtocolVersion, ok = in.ReadInt32()
	if !ok {
		return
	}
	var numUnrecognizedOptions int32
	numUnrecognizedOptions, ok = in.ReadInt32()
	if !ok {
		return
	}
	unrecognizedOptions = make([]string, 0, numUnrecognizedOptions)
	for i := 0; i < int(numUnrecognizedOptions); i++ {
		var unrecognizedOption string
		unrecognizedOption, ok = in.ReadString()
		if !ok {
			return
		}
		unrecognizedOptions = append(unrecognizedOptions, unrecognizedOption)
	}
	ok = true
	return
}

func WriteNegotiateProtocolVersion(out *zap.Packet, minorProtocolVersion int32, unrecognizedOptions []string) {
	out.WriteType(NegotiateProtocolVersion)
	out.WriteInt32(minorProtocolVersion)
	out.WriteInt32(int32(len(unrecognizedOptions)))
	for _, option := range unrecognizedOptions {
		out.WriteString(option)
	}
}
