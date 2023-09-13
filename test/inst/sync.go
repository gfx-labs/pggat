package inst

type Sync struct{}

func (Sync) instruction() {}

var _ Instruction = Sync{}
