package tests

import (
	"pggat/test"
	"pggat/test/inst"
)

var EQP0 = test.Test{
	Name: "EQP0",
	Instructions: []inst.Instruction{
		inst.Parse{
			Destination: "c",
			Query:       "select 1",
		},
		inst.Sync{},
		inst.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		inst.Parse{
			Destination: "b",
			Query:       "this is a bad query",
		},
		inst.Parse{
			Destination: "c",
			Query:       "select 1",
		},
		inst.Sync{},
		inst.DescribePreparedStatement("c"),
		inst.Sync{},
	},
}

var EQP1 = test.Test{
	Name: "EQP1",
	Instructions: []inst.Instruction{
		inst.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		inst.Parse{
			Destination: "b",
			Query:       "this is a bad query",
		},
		inst.Parse{
			Destination: "c",
			Query:       "select 1",
		},
		inst.Sync{},
		inst.DescribePreparedStatement("c"),
		inst.Sync{},
	},
}

var EQP2 = test.Test{
	Name: "EQP2",
	Instructions: []inst.Instruction{
		inst.Parse{
			Query: "select 0",
		},
		inst.Bind{
			Destination: "a",
		},
		inst.Sync{},
		inst.DescribePortal("a"),
		inst.Sync{},
	},
}

var EQP3 = test.Test{
	Name: "EQP3",
	Instructions: []inst.Instruction{
		inst.SimpleQuery("BEGIN"),
		inst.Parse{
			Query: "select 0",
		},
		inst.Bind{
			Destination: "a",
		},
		inst.Sync{},
		inst.DescribePortal("a"),
		inst.Sync{},
		inst.SimpleQuery("END"),
	},
}

var EQP4 = test.Test{
	Name: "EQP4",
	Instructions: []inst.Instruction{
		inst.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		inst.Parse{
			Destination: "b",
			Query:       "this is a bad query",
		},
		inst.ClosePreparedStatement("a"),
		inst.Sync{},
		inst.DescribePreparedStatement("a"),
		inst.Sync{},
	},
}

var EQP5 = test.Test{
	Name: "EQP5",
	Instructions: []inst.Instruction{
		inst.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		inst.Sync{},
		inst.ClosePreparedStatement("a"),
		inst.Sync{},
		inst.DescribePreparedStatement("a"),
		inst.Sync{},
	},
}

var EQP6 = test.Test{
	Name: "EQP6",
	Instructions: []inst.Instruction{
		inst.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		inst.Parse{
			Destination: "a",
			Query:       "select 1",
		},
		inst.Sync{},
		inst.DescribePreparedStatement("a"),
		inst.Sync{},
	},
}

var EQP7 = test.Test{
	Name: "EQP7",
	Instructions: []inst.Instruction{
		inst.SimpleQuery("BEGIN"),
		inst.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		inst.Bind{
			Destination: "a",
			Source:      "a",
		},
		inst.Bind{
			Destination: "b",
			Source:      "a",
		},
		inst.Sync{},
		inst.DescribePortal("a"),
		inst.DescribePreparedStatement("a"),
		inst.Sync{},
		inst.ClosePreparedStatement("a"),
		inst.Sync{},
		inst.DescribePortal("a"),
		inst.DescribePortal("b"),
		inst.Sync{},
		inst.SimpleQuery("END"),
	},
}
