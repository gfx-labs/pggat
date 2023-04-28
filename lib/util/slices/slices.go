package slices

func Prepend[T any](slice []T, elem T) []T {
	slice = append(slice, *new(T))
	copy(slice[1:], slice)
	slice[0] = elem
	return slice
}

func Remove[T any](slice []T, idx int) []T {
	copy(slice[idx:], slice[idx+1:])
	return slice[:len(slice)-1]
}

func Pop[T any](slice []T) ([]T, T) {
	if len(slice) == 0 {
		return slice, *new(T)
	}
	elem := slice[len(slice)-1]
	return slice[:len(slice)-1], elem
}

func Contains[T comparable](slice []T, value T) bool {
	for _, elem := range slice {
		if elem == value {
			return true
		}
	}
	return false
}
