package inst

type ClosePreparedStatement string

func (ClosePreparedStatement) instruction() {}

var _ Instruction = ClosePreparedStatement("")

type ClosePortal string

func (ClosePortal) instruction() {}

var _ Instruction = ClosePortal("")
