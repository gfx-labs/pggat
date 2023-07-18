package packets

import (
	"pggat2/lib/zap"
)

func ReadBind(in zap.Inspector) (destination string, source string, parameterFormatCodes []int16, parameterValues [][]byte, resultFormatCodes []int16, ok bool) {
	in.Reset()
	if in.Type() != Bind {
		return
	}
	destination, ok = in.String()
	if !ok {
		return
	}
	source, ok = in.String()
	if !ok {
		return
	}
	var parameterFormatCodesLength uint16
	parameterFormatCodesLength, ok = in.Uint16()
	if !ok {
		return
	}
	parameterFormatCodes = make([]int16, 0, int(parameterFormatCodesLength))
	for i := 0; i < int(parameterFormatCodesLength); i++ {
		var parameterFormatCode int16
		parameterFormatCode, ok = in.Int16()
		if !ok {
			return
		}
		parameterFormatCodes = append(parameterFormatCodes, parameterFormatCode)
	}
	var parameterValuesLength uint16
	parameterValuesLength, ok = in.Uint16()
	if !ok {
		return
	}
	parameterValues = make([][]byte, 0, int(parameterValuesLength))
	for i := 0; i < int(parameterValuesLength); i++ {
		var parameterValueLength int32
		parameterValueLength, ok = in.Int32()
		if !ok {
			return
		}
		var parameterValue []byte
		if parameterValueLength >= 0 {
			parameterValue = make([]byte, int(parameterValueLength))
			ok = in.Bytes(parameterValue)
			if !ok {
				return
			}
		}
		parameterValues = append(parameterValues, parameterValue)
	}
	var resultFormatCodesLength uint16
	resultFormatCodesLength, ok = in.Uint16()
	if !ok {
		return
	}
	resultFormatCodes = make([]int16, 0, int(resultFormatCodesLength))
	for i := 0; i < int(resultFormatCodesLength); i++ {
		var resultFormatCode int16
		resultFormatCode, ok = in.Int16()
		if !ok {
			return
		}
		resultFormatCodes = append(resultFormatCodes, resultFormatCode)
	}
	return
}

func WriteBind(out zap.Builder, destination, source string, parameterFormatCodes []int16, parameterValues [][]byte, resultFormatCodes []int16) {
	out.Reset()
	out.Type(Bind)
	out.String(destination)
	out.String(source)
	out.Uint16(uint16(len(parameterFormatCodes)))
	for _, v := range parameterFormatCodes {
		out.Int16(v)
	}
	out.Uint16(uint16(len(parameterValues)))
	for _, v := range parameterValues {
		if v == nil {
			out.Int32(-1)
			continue
		}
		out.Int32(int32(len(v)))
		out.Bytes(v)
	}
	out.Uint16(uint16(len(resultFormatCodes)))
	for _, v := range resultFormatCodes {
		out.Int16(v)
	}
}
