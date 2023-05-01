package rob

// Constraints is a bitfield used to control which Sink a job runs on.
// They can be declared by using const ... rob.Constraints = 1 << iota.
// Because Constraints is an int64, you may have a maximum of 64 constraints
//
// Example:
/*
	const (
		ConstraintOne rob.Constraints = 1 << iota
		ConstraintTwo
		ConstraintThree
	)

	var All = rob.Constraints.All(
		ConstraintOne,
		ConstraintTwo,
		ConstraintThree,
	)
*/
type Constraints int64

func (T Constraints) All(cn ...Constraints) Constraints {
	v := T
	for _, c := range cn {
		v |= c
	}
	return v
}

func (T Constraints) Satisfies(other Constraints) bool {
	return (other & T) == other
}
