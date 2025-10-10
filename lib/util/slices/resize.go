package slices

func Resize[T any](slice []T, length int) []T {
	switch {
	case cap(slice) < length:
		a := make([]T, length)
		copy(a, slice)
		return a
	case len(slice) < length:
		for len(slice) < length {
			slice = append(slice, *new(T))
		}
		return slice
	default:
		return slice[:length]
	}
}
