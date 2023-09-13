package inst

type Execute string

func (Execute) instruction() {}

var _ Instruction = Execute("")
