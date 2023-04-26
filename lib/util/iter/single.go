package iter

func Single[T any](value T) Iter[T] {
	ok := true
	return func() (T, bool) {
		if ok {
			ok = false
			return value, true
		}
		return *new(T), false
	}
}
