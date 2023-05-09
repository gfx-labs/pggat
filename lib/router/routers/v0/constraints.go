package routers

import "pggat2/lib/rob"

const (
	writeConstraint rob.Constraints = 1 << iota
)

func constraints(write bool) rob.Constraints {
	var c rob.Constraints
	if write {
		c = rob.Constraints.All(
			c,
			writeConstraint,
		)
	}
	return c
}
