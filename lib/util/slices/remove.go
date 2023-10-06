package slices

// Remove will remove the item from the slice, retaining the original order
func Remove[T comparable](slice []T, item T) []T {
	i := Index(slice, item)
	if i == -1 {
		return slice
	}
	return RemoveIndex(slice, i)
}

func RemoveIndex[T any](slice []T, idx int) []T {
	copy(slice[idx:], slice[idx+1:])
	slice[len(slice)-1] = *new(T)
	return slice[:len(slice)-1]
}
