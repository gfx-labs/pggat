package tests

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/test"
)

var Sync = test.Test{
	Name: "Sync",
	Packets: []fed.Packet{
		&packets.Sync{},
		MakeQuery("BEGIN;"),
		&packets.Sync{},
		MakeQuery("END;"),
	},
}
