package inst

type CopyData []byte

func (CopyData) instruction() {}

var _ Instruction = CopyData{}
