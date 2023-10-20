package tests

import (
	"gfx.cafe/gfx/pggat/lib/fed"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/test"
)

var EQP0 = test.Test{
	Name: "EQP0",
	Packets: []fed.Packet{
		&packets.Parse{
			Destination: "c",
			Query:       "select 1",
		},
		&packets.Sync{},
		&packets.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		&packets.Parse{
			Destination: "b",
			Query:       "this is a bad query",
		},
		&packets.Parse{
			Destination: "c",
			Query:       "select 1",
		},
		&packets.Sync{},
		&packets.Describe{Which: 'S', Name: "c"},
		&packets.Sync{},
	},
}

var EQP1 = test.Test{
	Name: "EQP1",
	Packets: []fed.Packet{
		&packets.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		&packets.Parse{
			Destination: "b",
			Query:       "this is a bad query",
		},
		&packets.Parse{
			Destination: "c",
			Query:       "select 1",
		},
		&packets.Sync{},
		&packets.Describe{Which: 'S', Name: "c"},
		&packets.Sync{},
	},
}

var EQP2 = test.Test{
	Name: "EQP2",
	Packets: []fed.Packet{
		&packets.Parse{
			Query: "select 0",
		},
		&packets.Bind{
			Destination: "a",
		},
		&packets.Sync{},
		&packets.Describe{Which: 'P', Name: "a"},
		&packets.Sync{},
	},
}

var EQP3 = test.Test{
	Name: "EQP3",
	Packets: []fed.Packet{
		MakeQuery("BEGIN"),
		&packets.Parse{
			Query: "select 0",
		},
		&packets.Bind{
			Destination: "a",
		},
		&packets.Sync{},
		&packets.Describe{Which: 'P', Name: "a"},
		&packets.Sync{},
		MakeQuery("END"),
	},
}

var EQP4 = test.Test{
	Name: "EQP4",
	Packets: []fed.Packet{
		&packets.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		&packets.Parse{
			Destination: "b",
			Query:       "this is a bad query",
		},
		&packets.Close{Which: 'S', Name: "a"},
		&packets.Sync{},
		&packets.Describe{Which: 'S', Name: "a"},
		&packets.Sync{},
	},
}

var EQP5 = test.Test{
	Name: "EQP5",
	Packets: []fed.Packet{
		&packets.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		&packets.Sync{},
		&packets.Close{Which: 'S', Name: "a"},
		&packets.Sync{},
		&packets.Describe{Which: 'S', Name: "a"},
		&packets.Sync{},
	},
}

var EQP6 = test.Test{
	Name: "EQP6",
	Packets: []fed.Packet{
		&packets.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		&packets.Parse{
			Destination: "a",
			Query:       "select 1",
		},
		&packets.Sync{},
		&packets.Describe{Which: 'S', Name: "a"},
		&packets.Sync{},
	},
}

var EQP7 = test.Test{
	Name: "EQP7",
	Packets: []fed.Packet{
		MakeQuery("BEGIN"),
		&packets.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		&packets.Bind{
			Destination: "a",
			Source:      "a",
		},
		&packets.Bind{
			Destination: "b",
			Source:      "a",
		},
		&packets.Sync{},
		&packets.Describe{Which: 'P', Name: "a"},
		&packets.Describe{Which: 'S', Name: "a"},
		&packets.Sync{},
		&packets.Close{Which: 'S', Name: "a"},
		&packets.Sync{},
		&packets.Describe{Which: 'P', Name: "a"},
		&packets.Describe{Which: 'P', Name: "b"},
		&packets.Sync{},
		MakeQuery("END"),
	},
}

var EQP8 = test.Test{
	Name: "EQP8",
	Packets: []fed.Packet{
		MakeQuery("BEGIN"),
		&packets.Parse{
			Destination: "a",
			Query:       "select 0",
		},
		&packets.Bind{
			Destination: "a",
			Source:      "a",
		},
		&packets.Sync{},
		&packets.Describe{Which: 'P', Name: "a"},
		&packets.Close{Which: 'P', Name: "a"},
		&packets.Describe{Which: 'P', Name: "a"},
		&packets.Sync{},
		&packets.Describe{Which: 'P', Name: "a"},
		&packets.Sync{},
		MakeQuery("END"),
	},
}
