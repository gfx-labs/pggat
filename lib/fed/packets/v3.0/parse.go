package packets

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Parse struct {
	Destination        string
	Query              string
	ParameterDataTypes []int32
}

func (T *Parse) ReadFromPacket(packet fed.Packet) bool {
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

func (T *Parse) IntoPacket(packet fed.Packet) fed.Packet {
	packet = packet.Reset(TypeParse, len(T.Destination)+len(T.Query)+4+len(T.ParameterDataTypes)*4)
	packet = packet.AppendString(T.Destination)
	packet = packet.AppendString(T.Query)
	packet = packet.AppendInt16(int16(len(T.ParameterDataTypes)))
	for _, v := range T.ParameterDataTypes {
		packet = packet.AppendInt32(v)
	}
	return packet
}
