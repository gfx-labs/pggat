package inst

type Bind struct {
	Destination string
	Source      string
}

func (Bind) instruction() {}

var _ Instruction = Bind{}
