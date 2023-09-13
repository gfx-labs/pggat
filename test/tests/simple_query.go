package tests

import (
	"pggat/test"
	"pggat/test/inst"
)

var SimpleQuery = test.Test{
	Name: "Simple Query",
	Instructions: []inst.Instruction{
		inst.SimpleQuery("select 1;"),
		inst.SimpleQuery("SELECT 2, 3, 4;"),
		inst.SimpleQuery("akfdfsjkfds;"),
	},
}
