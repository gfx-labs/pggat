package packets

import (
	"pggat2/lib/util/slices"
	"pggat2/lib/zap"
)

type NegotiateProtocolVersion struct {
	MinorProtocolVersion int32
	UnrecognizedOptions  []string
}

func (T *NegotiateProtocolVersion) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeNegotiateProtocolVersion {
		return false
	}
	p := packet.ReadInt32(&T.MinorProtocolVersion)

	var numUnrecognizedOptions int32
	p = p.ReadInt32(&numUnrecognizedOptions)

	T.UnrecognizedOptions = slices.Resize(T.UnrecognizedOptions, int(numUnrecognizedOptions))
	for i := 0; i < int(numUnrecognizedOptions); i++ {
		p = p.ReadString(&T.UnrecognizedOptions[i])
	}

	return true
}

func (T *NegotiateProtocolVersion) IntoPacket() zap.Packet {
	size := 8
	for _, v := range T.UnrecognizedOptions {
		size += len(v) + 1
	}

	packet := zap.NewPacket(TypeNegotiateProtocolVersion, size)
	packet = packet.AppendInt32(T.MinorProtocolVersion)
	packet = packet.AppendInt32(int32(len(T.UnrecognizedOptions)))
	for _, v := range T.UnrecognizedOptions {
		packet = packet.AppendString(v)
	}

	return packet
}
