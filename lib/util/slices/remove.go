package slices

func Remove[T comparable](slice []T, item T) []T {
	for i, s := range slice {
		if s == item {
			copy(slice[i:], slice[i+1:])
			return slice[:len(slice)-1]
		}
	}

	return slice
}
