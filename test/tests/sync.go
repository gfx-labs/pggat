package tests

import (
	"pggat/test"
	"pggat/test/inst"
)

var Sync = test.Test{
	Instructions: []inst.Instruction{
		inst.Sync{},
		inst.SimpleQuery("BEGIN;"),
		inst.Sync{},
		inst.SimpleQuery("END;"),
	},
}
