package packets

import "pggat2/lib/zap"

func ReadBind(in zap.ReadablePacket) (destination string, source string, parameterFormatCodes []int16, parameterValues [][]byte, resultFormatCodes []int16, ok bool) {
	if in.ReadType() != Bind {
		return
	}
	destination, ok = in.ReadString()
	if !ok {
		return
	}
	source, ok = in.ReadString()
	if !ok {
		return
	}
	var parameterFormatCodesLength uint16
	parameterFormatCodesLength, ok = in.ReadUint16()
	if !ok {
		return
	}
	parameterFormatCodes = make([]int16, 0, int(parameterFormatCodesLength))
	for i := 0; i < int(parameterFormatCodesLength); i++ {
		var parameterFormatCode int16
		parameterFormatCode, ok = in.ReadInt16()
		if !ok {
			return
		}
		parameterFormatCodes = append(parameterFormatCodes, parameterFormatCode)
	}
	var parameterValuesLength uint16
	parameterValuesLength, ok = in.ReadUint16()
	if !ok {
		return
	}
	parameterValues = make([][]byte, 0, int(parameterValuesLength))
	for i := 0; i < int(parameterValuesLength); i++ {
		var parameterValueLength int32
		parameterValueLength, ok = in.ReadInt32()
		if !ok {
			return
		}
		var parameterValue []byte
		if parameterValueLength >= 0 {
			parameterValue = make([]byte, int(parameterValueLength))
			ok = in.ReadBytes(parameterValue)
			if !ok {
				return
			}
		}
		parameterValues = append(parameterValues, parameterValue)
	}
	var resultFormatCodesLength uint16
	resultFormatCodesLength, ok = in.ReadUint16()
	if !ok {
		return
	}
	resultFormatCodes = make([]int16, 0, int(resultFormatCodesLength))
	for i := 0; i < int(resultFormatCodesLength); i++ {
		var resultFormatCode int16
		resultFormatCode, ok = in.ReadInt16()
		if !ok {
			return
		}
		resultFormatCodes = append(resultFormatCodes, resultFormatCode)
	}
	return
}

func WriteBind(out *zap.Packet, destination, source string, parameterFormatCodes []int16, parameterValues [][]byte, resultFormatCodes []int16) {
	out.WriteType(Bind)
	out.WriteString(destination)
	out.WriteString(source)
	out.WriteUint16(uint16(len(parameterFormatCodes)))
	for _, v := range parameterFormatCodes {
		out.WriteInt16(v)
	}
	out.WriteUint16(uint16(len(parameterValues)))
	for _, v := range parameterValues {
		if v == nil {
			out.WriteInt32(-1)
			continue
		}
		out.WriteInt32(int32(len(v)))
		out.WriteBytes(v)
	}
	out.WriteUint16(uint16(len(resultFormatCodes)))
	for _, v := range resultFormatCodes {
		out.WriteInt16(v)
	}
}
