package test

import (
	"gfx.cafe/gfx/pggat/lib/fed"
)

type Test struct {
	// SideEffects determines whether this test has side effects such as creating or dropping tables.
	// This will prevent fail and stress testing, as those would immediately fail.
	SideEffects bool

	Name    string
	Packets []fed.Packet
}
