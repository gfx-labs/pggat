package tests

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/test"
)

var CopyOut0 = test.Test{
	SideEffects: true,
	Name:        "Copy Out 0",
	Packets: []fed.Packet{
		MakeQuery("CREATE TABLE test ( x integer NOT NULL, y varchar(40) NOT NULL PRIMARY KEY )"),
		MakeQuery("INSERT INTO test VALUES (123, 'hello world')"),
		MakeQuery("INSERT INTO test VALUES (-324, 'garet was here')"),
		MakeQuery("COPY test TO STDOUT"),
		MakeQuery("DROP TABLE test"),
	},
}

var CopyOut1 = test.Test{
	SideEffects: true,
	Name:        "Copy Out 1",
	Packets: []fed.Packet{
		MakeQuery("CREATE TABLE test ( x integer NOT NULL, y varchar(40) NOT NULL PRIMARY KEY )"),
		MakeQuery("INSERT INTO test VALUES (123, 'hello world')"),
		MakeQuery("INSERT INTO test VALUES (-324, 'garet was here')"),
		&packets.Parse{
			Query: "COPY test TO STDOUT",
		},
		&packets.Describe{
			Which: 'S',
		},
		&packets.Bind{},
		&packets.Describe{
			Which: 'P',
		},
		&packets.Execute{},
		&packets.Sync{},
		MakeQuery("DROP TABLE test"),
	},
}
