package iter

func Filter[T any](iter Iter[T], f func(T) bool) Iter[T] {
	return func() (T, bool) {
		for v, ok := iter(); ok; v, ok = iter() {
			if f(v) {
				return v, true
			}
		}
		return *new(T), false
	}
}
