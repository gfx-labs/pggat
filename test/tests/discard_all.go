package tests

import (
	"gfx.cafe/gfx/pggat/test"
	"gfx.cafe/gfx/pggat/test/inst"
)

var DiscardAll = test.Test{
	Name: "Discard All",
	Instructions: []inst.Instruction{
		inst.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		inst.Bind{
			Destination: "a",
			Source:      "a",
		},
		inst.Sync{},
		inst.SimpleQuery("discard all"),
		inst.DescribePreparedStatement("a"),
		inst.Sync{},
		inst.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		inst.DescribePreparedStatement("a"),
		inst.Sync{},
	},
}
