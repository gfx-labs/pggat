package frontends

type PreparedStatement struct {
	Query              string
	ParameterDataTypes []int32
}
