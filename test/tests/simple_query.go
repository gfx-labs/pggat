package tests

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/test"
)

func MakeQuery(query string) *packets.Query {
	return (*packets.Query)(&query)
}

var SimpleQuery = test.Test{
	Name: "Simple Query",
	Packets: []fed.Packet{
		MakeQuery("select 1;"),
		MakeQuery("SELECT 2, 3, 4;"),
		MakeQuery("akfdfsjkfds;"),
	},
}
