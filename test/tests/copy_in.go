package tests

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/test"
)

var CopyIn0 = test.Test{
	SideEffects: true,
	Name:        "Copy In 0",
	Packets: []fed.Packet{
		MakeQuery("CREATE TABLE test ( x integer NOT NULL, y varchar(40) NOT NULL PRIMARY KEY )"),
		MakeQuery("COPY test FROM STDIN"),
		&packets.CopyData{49, 50, 51, 9, 104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 10},
		&packets.CopyData{45, 51, 50, 52, 9, 103, 97, 114, 101, 116, 32, 119, 97, 115, 32, 104, 101, 114, 101, 10},
		&packets.CopyDone{},
		MakeQuery("DROP TABLE test"),
	},
}

var CopyIn1 = test.Test{
	SideEffects: true,
	Name:        "Copy In 1",
	Packets: []fed.Packet{
		MakeQuery("CREATE TABLE test ( x integer NOT NULL, y varchar(40) NOT NULL PRIMARY KEY )"),
		&packets.Parse{
			Query: "COPY test FROM STDIN",
		},
		&packets.Describe{
			Which: 'S',
			Name:  "",
		},
		&packets.Bind{},
		&packets.Describe{
			Which: 'P',
			Name:  "",
		},
		&packets.Execute{},
		&packets.Sync{},
		&packets.CopyData{49, 50, 51, 9, 104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 10},
		&packets.CopyData{45, 51, 50, 52, 9, 103, 97, 114, 101, 116, 32, 119, 97, 115, 32, 104, 101, 114, 101, 10},
		&packets.CopyDone{},
		&packets.Sync{},
		MakeQuery("DROP TABLE test"),
	},
}
