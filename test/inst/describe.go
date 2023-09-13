package inst

type DescribePortal string

func (DescribePortal) instruction() {}

var _ Instruction = DescribePortal("")

type DescribePreparedStatement string

func (DescribePreparedStatement) instruction() {}

var _ Instruction = DescribePreparedStatement("")
