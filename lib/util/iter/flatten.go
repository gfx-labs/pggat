package iter

func Flatten[T any](iter Iter[Iter[T]]) Iter[T] {
	i := Empty[T]()
	return func() (T, bool) {
		for {
			v, ok := i()
			if ok {
				return v, true
			}
			i, ok = iter()
			if !ok {
				break
			}
		}
		return *new(T), false
	}
}
