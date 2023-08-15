package slices

func Contains[T comparable](haystack []T, needle T) bool {
	for _, hay := range haystack {
		if hay == needle {
			return true
		}
	}

	return false
}
