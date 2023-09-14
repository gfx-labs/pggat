package tests

import (
	"pggat/test"
	"pggat/test/inst"
)

var CopyIn0 = test.Test{
	SideEffects: true,
	Name:        "Copy In 0",
	Instructions: []inst.Instruction{
		inst.SimpleQuery("CREATE TABLE test ( x integer NOT NULL, y varchar(40) NOT NULL PRIMARY KEY )"),
		inst.SimpleQuery("COPY test FROM STDIN"),
		inst.CopyData{49, 50, 51, 9, 104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 10},
		inst.CopyData{45, 51, 50, 52, 9, 103, 97, 114, 101, 116, 32, 119, 97, 115, 32, 104, 101, 114, 101, 10},
		inst.CopyDone{},
		inst.SimpleQuery("DROP TABLE test"),
	},
}

var CopyIn1 = test.Test{
	SideEffects: true,
	Name:        "Copy In 1",
	Instructions: []inst.Instruction{
		inst.SimpleQuery("CREATE TABLE test ( x integer NOT NULL, y varchar(40) NOT NULL PRIMARY KEY )"),
		inst.Parse{
			Query: "COPY test FROM STDIN",
		},
		inst.DescribePreparedStatement(""),
		inst.Bind{},
		inst.DescribePortal(""),
		inst.Execute(""),
		inst.Sync{},
		inst.CopyData{49, 50, 51, 9, 104, 101, 108, 108, 111, 32, 119, 111, 114, 108, 100, 10},
		inst.CopyData{45, 51, 50, 52, 9, 103, 97, 114, 101, 116, 32, 119, 97, 115, 32, 104, 101, 114, 101, 10},
		inst.CopyDone{},
		inst.SimpleQuery("DROP TABLE test"),
	},
}
