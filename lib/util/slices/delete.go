package slices

// Delete is similar to Remove but doesn't retain order.
func Delete[T comparable](slice []T, item T) []T {
	i := Index(slice, item)
	if i == -1 {
		return slice
	}
	return DeleteIndex(slice, i)
}

func DeleteIndex[T any](slice []T, idx int) []T {
	slice[idx] = slice[len(slice)-1]
	slice[len(slice)-1] = *new(T)
	return slice[:len(slice)-1]
}
