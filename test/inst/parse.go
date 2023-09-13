package inst

type Parse struct {
	Destination string
	Query       string
}

func (Parse) instruction() {}

var _ Instruction = Parse{}
