package rob

type Source interface {
	// Schedule work with constraints. Work will run on a Sink that at least fulfills these constraints
	Schedule(work any, constraints Constraints)
}
