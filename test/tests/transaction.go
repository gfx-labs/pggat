package tests

import (
	"pggat/test"
	"pggat/test/inst"
)

var Transaction = test.Test{
	Instructions: []inst.Instruction{
		inst.SimpleQuery("BEGIN;"),
		inst.SimpleQuery("select 1;"),
		inst.SimpleQuery("this will fail;"),
		inst.SimpleQuery("select 2;"),
		inst.SimpleQuery("END;"),
	},
}
