package packets

import (
	"pggat2/lib/fed"
	"pggat2/lib/util/slices"
)

type Bind struct {
	Destination          string
	Source               string
	ParameterFormatCodes []int16
	ParameterValues      [][]byte
	ResultFormatCodes    []int16
}

func (T *Bind) ReadFromPacket(packet fed.Packet) bool {
	if packet.Type() != TypeBind {
		return false
	}
	p := packet.ReadString(&T.Destination)
	p = p.ReadString(&T.Source)

	var parameterFormatCodesLength uint16
	p = p.ReadUint16(&parameterFormatCodesLength)
	T.ParameterFormatCodes = slices.Resize(T.ParameterFormatCodes, int(parameterFormatCodesLength))
	for i := 0; i < int(parameterFormatCodesLength); i++ {
		p = p.ReadInt16(&T.ParameterFormatCodes[i])
	}

	var parameterValuesLength uint16
	p = p.ReadUint16(&parameterValuesLength)
	T.ParameterValues = slices.Resize(T.ParameterValues, int(parameterValuesLength))
	for i := 0; i < int(parameterValuesLength); i++ {
		var parameterValueLength int32
		p = p.ReadInt32(&parameterValueLength)
		if parameterValueLength == -1 {
			T.ParameterValues[i] = nil
			continue
		}
		T.ParameterValues[i] = slices.Resize(T.ParameterValues[i], int(parameterValueLength))
		p = p.ReadBytes(T.ParameterValues[i])
	}

	var resultFormatCodesLength uint16
	p = p.ReadUint16(&resultFormatCodesLength)
	T.ResultFormatCodes = slices.Resize(T.ResultFormatCodes, int(resultFormatCodesLength))
	for i := 0; i < int(resultFormatCodesLength); i++ {
		p = p.ReadInt16(&T.ResultFormatCodes[i])
	}

	return true
}

func (T *Bind) IntoPacket() fed.Packet {
	size := 0
	size += len(T.Destination) + 1
	size += len(T.Source) + 1
	size += 2
	size += len(T.ParameterFormatCodes) * 2
	size += 2
	for _, v := range T.ParameterValues {
		size += 4 + len(v)
	}
	size += 2
	size += len(T.ResultFormatCodes) * 2

	packet := fed.NewPacket(TypeBind, size)
	packet = packet.AppendString(T.Destination)
	packet = packet.AppendString(T.Source)
	packet = packet.AppendUint16(uint16(len(T.ParameterFormatCodes)))
	for _, v := range T.ParameterFormatCodes {
		packet = packet.AppendInt16(v)
	}
	packet = packet.AppendUint16(uint16(len(T.ParameterValues)))
	for _, v := range T.ParameterValues {
		if v == nil {
			packet = packet.AppendInt32(-1)
			continue
		}
		packet = packet.AppendInt32(int32(len(v)))
		packet = packet.AppendBytes(v)
	}
	packet = packet.AppendUint16(uint16(len(T.ResultFormatCodes)))
	for _, v := range T.ResultFormatCodes {
		packet = packet.AppendInt16(v)
	}
	return packet
}
