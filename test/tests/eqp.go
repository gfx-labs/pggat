package tests

import (
	"pggat/test"
	"pggat/test/inst"
)

var EQP = test.Test{
	Name: "EQP",
	Instructions: []inst.Instruction{
		inst.Parse{
			Destination: "test",
			Query:       "select 0;",
		},
		inst.DescribePreparedStatement("test"),
		inst.Bind{
			Destination: "test",
			Source:      "test",
		},
		inst.DescribePortal("test"),
		inst.Execute("test"),
		inst.Sync{},
	},
}
