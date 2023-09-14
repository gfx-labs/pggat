package inst

type CopyDone struct{}

func (CopyDone) instruction() {}

var _ Instruction = CopyDone{}
