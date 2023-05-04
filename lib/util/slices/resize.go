package slices

func Resize[T any](slice []T, length int) []T {
	if cap(slice) < length {
		return make([]T, length)
	} else if len(slice) < length {
		for len(slice) < length {
			slice = append(slice, *new(T))
		}
		return slice
	} else {
		return slice[:length]
	}
}
