package slices

func Equal[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}

	for i, av := range a {
		bv := b[i]
		if av != bv {
			return false
		}
	}

	return true
}
