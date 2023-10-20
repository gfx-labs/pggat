package tests

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/test"
)

var DiscardAll = test.Test{
	Name: "Discard All",
	Packets: []fed.Packet{
		&packets.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		&packets.Bind{
			Destination: "a",
			Source:      "a",
		},
		&packets.Sync{},
		MakeQuery("SET application_name = 'test_application'"),
		MakeQuery("SHOW application_name"),
		MakeQuery("discard all"),
		MakeQuery("SHOW application_name"),
		&packets.Describe{
			Which: 'S',
			Name:  "a",
		},
		&packets.Sync{},
		&packets.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		&packets.Describe{
			Which: 'S',
			Name:  "a",
		},
		&packets.Sync{},
	},
}
