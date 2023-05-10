package packets

import "pggat2/lib/pnet/packet"

func ReadParse(in packet.In) (destination string, query string, parameterDataTypes []int32, ok bool) {
	in.Reset()
	if in.Type() != packet.Parse {
		return
	}
	destination, ok = in.String()
	if !ok {
		return
	}
	query, ok = in.String()
	if !ok {
		return
	}
	var parameterDataTypesCount int16
	parameterDataTypesCount, ok = in.Int16()
	if !ok {
		return
	}
	parameterDataTypes = make([]int32, 0, int(parameterDataTypesCount))
	for i := 0; i < int(parameterDataTypesCount); i++ {
		var parameterDataType int32
		parameterDataType, ok = in.Int32()
		if !ok {
			return
		}
		parameterDataTypes = append(parameterDataTypes, parameterDataType)
	}
	return
}

func WriteParse(out packet.Out, destination string, query string, parameterDataTypes []int32) {
	out.Reset()
	out.Type(packet.Parse)
	out.String(destination)
	out.String(query)
	out.Int16(int16(len(parameterDataTypes)))
	for _, v := range parameterDataTypes {
		out.Int32(v)
	}
}
