package tests

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/test"
)

var Transaction = test.Test{
	Name: "Transaction",
	Packets: []fed.Packet{
		MakeQuery("BEGIN;"),
		MakeQuery("select 1;"),
		MakeQuery("this will fail;"),
		MakeQuery("select 2;"),
		MakeQuery("END;"),
	},
}
