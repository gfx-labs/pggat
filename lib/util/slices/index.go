package slices

func Index[T comparable](haystack []T, needle T) int {
	for i, v := range haystack {
		if needle == v {
			return i
		}
	}
	return -1
}
