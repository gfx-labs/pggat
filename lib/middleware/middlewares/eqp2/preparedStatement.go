package eqp2

import "pggat2/lib/util/slices"

type PreparedStatement struct {
	Query              []byte
	ParameterDataTypes []int32
}

func (T *PreparedStatement) Equals(rhs *PreparedStatement) bool {
	if T == rhs {
		return true
	}
	if !slices.Equal(T.Query, rhs.Query) {
		return false
	}
	if !slices.Equal(T.ParameterDataTypes, rhs.ParameterDataTypes) {
		return false
	}
	return true
}
