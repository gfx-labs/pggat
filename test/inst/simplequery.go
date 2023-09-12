package inst

type SimpleQuery string

func (SimpleQuery) instruction() {}

var _ Instruction = SimpleQuery("")
