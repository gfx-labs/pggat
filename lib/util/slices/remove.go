package slices

// Remove will check for item in the target slice. If it finds it, it will move it to the end of the slice and return a slice
// with length-1. The original slice will contain all items (though in a different order), and the new slice will contain all
// but item.
func Remove[T comparable](slice []T, item T) []T {
	for i, s := range slice {
		if s == item {
			copy(slice[i:], slice[i+1:])
			slice[len(slice)-1] = item
			return slice[:len(slice)-1]
		}
	}

	return slice
}
