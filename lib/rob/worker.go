package rob

type Worker interface {
	Do(constraints Constraints, work any)
}
