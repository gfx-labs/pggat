package eqp2

import "pggat2/lib/util/slices"

type Portal struct {
	Source               string
	ParameterFormatCodes []int16
	ParameterValues      [][]byte
	ResultFormatCodes    []int16
}

func (T *Portal) Equals(rhs *Portal) bool {
	if T == rhs {
		return true
	}
	if T.Source != rhs.Source {
		return false
	}
	if !slices.Equal(T.ParameterFormatCodes, rhs.ParameterFormatCodes) {
		return false
	}
	if len(T.ParameterValues) != len(rhs.ParameterValues) {
		return false
	}
	for i := range T.ParameterValues {
		if !slices.Equal(T.ParameterValues[i], rhs.ParameterValues[i]) {
			return false
		}
	}
	if !slices.Equal(T.ResultFormatCodes, rhs.ResultFormatCodes) {
		return false
	}
	return true
}
