package iter

func Empty[T any]() Iter[T] {
	return func() (T, bool) {
		return *new(T), false
	}
}
