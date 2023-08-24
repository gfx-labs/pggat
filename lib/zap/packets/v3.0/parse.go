package packets

import (
	"pggat2/lib/util/slices"
	"pggat2/lib/zap"
)

type Parse struct {
	Destination        string
	Query              string
	ParameterDataTypes []int32
}

func (T *Parse) ReadFromPacket(packet zap.Packet) bool {
	if packet.Type() != TypeParse {
		return false
	}
	p := packet.ReadString(&T.Destination)
	p = p.ReadString(&T.Query)
	var parameterDataTypesCount int16
	p = p.ReadInt16(&parameterDataTypesCount)
	T.ParameterDataTypes = slices.Resize(T.ParameterDataTypes, int(parameterDataTypesCount))
	for i := 0; i < int(parameterDataTypesCount); i++ {
		p = p.ReadInt32(&T.ParameterDataTypes[i])
	}
	return true
}

func (T *Parse) IntoPacket() zap.Packet {
	packet := zap.NewPacket(TypeParse)
	packet = packet.AppendString(T.Destination)
	packet = packet.AppendString(T.Query)
	packet = packet.AppendInt16(int16(len(T.ParameterDataTypes)))
	for _, v := range T.ParameterDataTypes {
		packet = packet.AppendInt32(v)
	}
	return packet
}
