package slices

func Resize[T any](slice []T, length int) []T {
	if cap(slice) < length {
		a := make([]T, length)
		copy(a, slice)
		return a
	} else if len(slice) < length {
		for len(slice) < length {
			slice = append(slice, *new(T))
		}
		return slice
	} else {
		return slice[:length]
	}
}
