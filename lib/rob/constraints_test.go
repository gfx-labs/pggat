package rob

import "testing"

const (
	ConstraintOne Constraints = 1 << iota
	ConstraintTwo
	ConstraintThree
	ConstraintFour
)

func TestConstraints_All(t *testing.T) {
	all := Constraints.All(ConstraintOne, ConstraintTwo, ConstraintThree, ConstraintFour)
	if all != 0b1111 {
		t.Error("expected all bits to be set")
	}
	odd := Constraints.All(ConstraintOne, ConstraintThree)
	if odd != 0b0101 {
		t.Error("expected odd bits to be set")
	}
	even := Constraints.All(ConstraintTwo, ConstraintFour)
	if even != 0b1010 {
		t.Error("expected even bits to be set")
	}
}

func TestConstraints_Satisfies(t *testing.T) {
	all := Constraints.All(ConstraintOne, ConstraintTwo, ConstraintThree, ConstraintFour)
	if ConstraintOne.Satisfies(all) {
		t.Error("expected one to not satisfy all")
	}
	odd := Constraints.All(ConstraintOne, ConstraintThree)
	if odd.Satisfies(all) {
		t.Error("expected odd to not satisfy all")
	}
	if ConstraintOne.Satisfies(odd) {
		t.Error("expected one to not satisfy odd")
	}
	even := Constraints.All(ConstraintTwo, ConstraintFour)
	if even.Satisfies(all) {
		t.Error("expected even to not satisfy all")
	}
	if ConstraintOne.Satisfies(even) {
		t.Error("expected one to not satisfy even")
	}
	if !even.Satisfies(even) {
		t.Error("expected even to satisfy even")
	}
	if !all.Satisfies(even) {
		t.Error("expected all to satisfy even")
	}
	if !all.Satisfies(odd) {
		t.Error("expected all to satisfy odd")
	}
}
