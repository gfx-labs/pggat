package test

import "pggat/test/inst"

type Test struct {
	// SideEffects determines whether this test has side effects such as creating or dropping tables.
	// This will prevent fail and stress testing, as those would immediately fail.
	SideEffects bool

	Name         string
	Instructions []inst.Instruction
}
