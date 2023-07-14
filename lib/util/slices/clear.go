package slices

func Clear[T any](slice []T) {
	for i := 0; i < len(slice); i++ {
		slice[i] = *new(T)
	}
}
