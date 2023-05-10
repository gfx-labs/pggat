package eqp

import "pggat2/lib/util/slices"

type PreparedStatement struct {
	Query              string
	ParameterDataTypes []int32
}

func (T PreparedStatement) Equals(rhs PreparedStatement) bool {
	if T.Query != rhs.Query {
		return false
	}
	if !slices.Equal(T.ParameterDataTypes, rhs.ParameterDataTypes) {
		return false
	}
	return true
}
