package eqp2

import (
	"pggat2/lib/util/pools"
	"pggat2/lib/util/slices"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type Pooler struct {
	uint8Slice        pools.Pool[[]byte]
	uint8SliceSlice   pools.Pool[[][]byte]
	int16Slice        pools.Pool[[]int16]
	int32Slice        pools.Pool[[]int32]
	portal            pools.Pool[*Portal]
	preparedStatement pools.Pool[*PreparedStatement]
}

func (T *Pooler) PutUint8Slice(v []byte) {
	if v == nil {
		return
	}
	T.uint8Slice.Put(v[:0])
}

func (T *Pooler) PutUint8SliceSlice(v [][]byte) {
	if v == nil {
		return
	}
	for _, b := range v {
		T.PutUint8Slice(b)
	}
	T.uint8SliceSlice.Put(v[:0])
}

func (T *Pooler) PutInt16Slice(v []int16) {
	if v == nil {
		return
	}
	T.int16Slice.Put(v[:0])
}

func (T *Pooler) PutInt32Slice(v []int32) {
	if v == nil {
		return
	}
	T.int32Slice.Put(v[:0])
}

func (T *Pooler) PutPortal(portal *Portal) {
	if portal == nil {
		return
	}
	T.PutInt16Slice(portal.ParameterFormatCodes)
	T.PutUint8SliceSlice(portal.ParameterValues)
	T.PutInt16Slice(portal.ResultFormatCodes)
	*portal = Portal{}
	T.portal.Put(portal)
}

func (T *Pooler) PutPreparedStatement(preparedStatement *PreparedStatement) {
	if preparedStatement == nil {
		return
	}
	T.PutUint8Slice(preparedStatement.Query)
	T.PutInt32Slice(preparedStatement.ParameterDataTypes)
	*preparedStatement = PreparedStatement{}
	T.preparedStatement.Put(preparedStatement)
}

func (T *Pooler) GetUint8Slice() []byte {
	v, _ := T.uint8Slice.Get()
	return v
}

func (T *Pooler) GetUint8SliceSlice() [][]byte {
	v, _ := T.uint8SliceSlice.Get()
	return v
}

func (T *Pooler) GetInt16Slice() []int16 {
	v, _ := T.int16Slice.Get()
	return v
}

func (T *Pooler) GetInt32Slice() []int32 {
	v, _ := T.int32Slice.Get()
	return v
}

func (T *Pooler) GetPortal() *Portal {
	v, ok := T.portal.Get()
	if !ok {
		v = &Portal{}
	}
	return v
}

func (T *Pooler) GetPreparedStatement() *PreparedStatement {
	v, ok := T.preparedStatement.Get()
	if !ok {
		v = &PreparedStatement{}
	}
	return v
}

func (T *Pooler) ClonePortal(portal *Portal) *Portal {
	clone := T.GetPortal()
	clone.Source = portal.Source
	clone.ParameterFormatCodes = slices.CloneInto(T.GetInt16Slice(), portal.ParameterFormatCodes)
	clone.ParameterValues = slices.Resize(T.GetUint8SliceSlice(), len(portal.ParameterValues))
	for i, v := range portal.ParameterValues {
		clone.ParameterValues[i] = slices.CloneInto(T.GetUint8Slice(), v)
	}
	clone.ResultFormatCodes = slices.CloneInto(T.GetInt16Slice(), portal.ResultFormatCodes)
	return clone
}

func (T *Pooler) ClonePreparedStatement(preparedStatement *PreparedStatement) *PreparedStatement {
	clone := T.GetPreparedStatement()
	clone.Query = slices.CloneInto(T.GetUint8Slice(), preparedStatement.Query)
	clone.ParameterDataTypes = slices.CloneInto(T.GetInt32Slice(), preparedStatement.ParameterDataTypes)
	return clone
}

func (T *Pooler) ReadBind(in zap.In) (destination string, portal *Portal, ok bool) {
	in.Reset()
	if in.Type() != packets.Bind {
		return
	}
	destination, ok = in.String()
	if !ok {
		return
	}
	portal = T.GetPortal()
	portal.Source, ok = in.String()
	if !ok {
		T.PutPortal(portal)
		portal = nil
		return
	}
	var parameterFormatCodesLength uint16
	parameterFormatCodesLength, ok = in.Uint16()
	if !ok {
		T.PutPortal(portal)
		portal = nil
		return
	}
	portal.ParameterFormatCodes = slices.Resize(T.GetInt16Slice(), int(parameterFormatCodesLength))
	for i := 0; i < int(parameterFormatCodesLength); i++ {
		portal.ParameterFormatCodes[i], ok = in.Int16()
		if !ok {
			T.PutPortal(portal)
			portal = nil
			return
		}
	}
	var parameterValuesLength uint16
	parameterValuesLength, ok = in.Uint16()
	if !ok {
		T.PutPortal(portal)
		portal = nil
		return
	}
	portal.ParameterValues = slices.Resize(T.GetUint8SliceSlice(), int(parameterValuesLength))
	for i := 0; i < int(parameterValuesLength); i++ {
		var parameterValueLength int32
		parameterValueLength, ok = in.Int32()
		if !ok {
			T.PutPortal(portal)
			portal = nil
			return
		}
		if parameterValueLength >= 0 {
			portal.ParameterValues[i] = slices.Resize(T.GetUint8Slice(), int(parameterValueLength))
			ok = in.Bytes(portal.ParameterValues[i])
			if !ok {
				T.PutPortal(portal)
				portal = nil
				return
			}
		}
	}
	var resultFormatCodesLength uint16
	resultFormatCodesLength, ok = in.Uint16()
	if !ok {
		T.PutPortal(portal)
		portal = nil
		return
	}
	portal.ResultFormatCodes = slices.Resize(T.GetInt16Slice(), int(resultFormatCodesLength))
	for i := 0; i < int(resultFormatCodesLength); i++ {
		portal.ResultFormatCodes[i], ok = in.Int16()
		if !ok {
			T.PutPortal(portal)
			portal = nil
			return
		}
	}
	return
}

func (T *Pooler) ReadParse(in zap.In) (destination string, preparedStatement *PreparedStatement, ok bool) {
	in.Reset()
	if in.Type() != packets.Parse {
		return "", nil, false
	}

	destination, ok = in.String()
	if !ok {
		return
	}
	preparedStatement = T.GetPreparedStatement()
	preparedStatement.Query, ok = in.StringBytes(T.GetUint8Slice())
	if !ok {
		T.PutPreparedStatement(preparedStatement)
		preparedStatement = nil
		return
	}
	var parameterDataTypesCount int16
	parameterDataTypesCount, ok = in.Int16()
	if !ok {
		T.PutPreparedStatement(preparedStatement)
		preparedStatement = nil
		return
	}
	preparedStatement.ParameterDataTypes = slices.Resize(T.GetInt32Slice(), int(parameterDataTypesCount))
	for i := 0; i < int(parameterDataTypesCount); i++ {
		preparedStatement.ParameterDataTypes[i], ok = in.Int32()
		if !ok {
			T.PutPreparedStatement(preparedStatement)
			preparedStatement = nil
			return
		}
	}
	return
}
