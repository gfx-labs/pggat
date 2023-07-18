package packets

import (
	"pggat2/lib/zap"
)

func ReadParse(in *zap.ReadablePacket) (destination string, query string, parameterDataTypes []int32, ok bool) {
	if in.ReadType() != Parse {
		return
	}
	destination, ok = in.ReadString()
	if !ok {
		return
	}
	query, ok = in.ReadString()
	if !ok {
		return
	}
	var parameterDataTypesCount int16
	parameterDataTypesCount, ok = in.ReadInt16()
	if !ok {
		return
	}
	parameterDataTypes = make([]int32, 0, int(parameterDataTypesCount))
	for i := 0; i < int(parameterDataTypesCount); i++ {
		var parameterDataType int32
		parameterDataType, ok = in.ReadInt32()
		if !ok {
			return
		}
		parameterDataTypes = append(parameterDataTypes, parameterDataType)
	}
	return
}

func WriteParse(out *zap.Packet, destination string, query string, parameterDataTypes []int32) {
	out.WriteType(Parse)
	out.WriteString(destination)
	out.WriteString(query)
	out.WriteInt16(int16(len(parameterDataTypes)))
	for _, v := range parameterDataTypes {
		out.WriteInt32(v)
	}
}
