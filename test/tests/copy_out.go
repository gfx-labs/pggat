package tests

import (
	"pggat/test"
	"pggat/test/inst"
)

var CopyOut0 = test.Test{
	Name: "Copy Out 0",
	Instructions: []inst.Instruction{
		inst.SimpleQuery("CREATE TABLE test ( x integer NOT NULL, y varchar(40) NOT NULL PRIMARY KEY )"),
		inst.SimpleQuery("INSERT INTO test VALUES (123, 'hello world')"),
		inst.SimpleQuery("INSERT INTO test VALUES (-324, 'garet was here')"),
		inst.SimpleQuery("COPY test TO STDOUT"),
		inst.SimpleQuery("DROP TABLE test"),
	},
}

var CopyOut1 = test.Test{
	Name: "Copy Out 1",
	Instructions: []inst.Instruction{
		inst.SimpleQuery("CREATE TABLE test ( x integer NOT NULL, y varchar(40) NOT NULL PRIMARY KEY )"),
		inst.SimpleQuery("INSERT INTO test VALUES (123, 'hello world')"),
		inst.SimpleQuery("INSERT INTO test VALUES (-324, 'garet was here')"),
		inst.Parse{
			Query: "COPY test TO STDOUT",
		},
		inst.DescribePreparedStatement(""),
		inst.Bind{},
		inst.DescribePortal(""),
		inst.Execute(""),
		inst.Sync{},
		inst.SimpleQuery("DROP TABLE test"),
	},
}
