package iter

func Slice[T any](slice []T) Iter[T] {
	i := 0
	return func() (T, bool) {
		if i >= len(slice) {
			return *new(T), false
		}
		v := slice[i]
		i++
		return v, true
	}
}
