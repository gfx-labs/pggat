package iter

func Map[T, U any](base Iter[T], f func(T) U) Iter[U] {
	return func() (U, bool) {
		v, ok := base()
		if !ok {
			return *new(U), false
		}
		return f(v), true
	}
}
