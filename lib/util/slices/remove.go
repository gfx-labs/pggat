package slices

// Remove will check for item in the target slice. If it finds it, it will move it to the end of the slice and return a slice
// with length-1. The original slice will contain all items (though in a different order), and the new slice will contain all
// but item.
func Remove[T comparable](slice []T, item T) []T {
	i := Index(slice, item)
	if i == -1 {
		return slice
	}
	return RemoveIndex(slice, i)
}

func RemoveIndex[T any](slice []T, idx int) []T {
	item := slice[idx]
	copy(slice[idx:], slice[idx+1:])
	slice[len(slice)-1] = item
	return slice[:len(slice)-1]
}

// Delete is similar to Remove but leaves a *new(T) in the old slice, allowing the value to be GC'd
func Delete[T comparable](slice []T, item T) []T {
	i := Index(slice, item)
	if i == -1 {
		return slice
	}
	return DeleteIndex(slice, i)
}

func DeleteIndex[T any](slice []T, idx int) []T {
	copy(slice[idx:], slice[idx+1:])
	slice[len(slice)-1] = *new(T)
	return slice[:len(slice)-1]
}
