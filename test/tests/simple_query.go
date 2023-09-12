package tests

import (
	"pggat/test"
	"pggat/test/inst"
)

var SimpleQuery = test.Test{
	Parallel: true,
	Instructions: []inst.Instruction{
		inst.SimpleQuery("select 1;"),
	},
}
