package iter

func chain[T any](i1, i2 Iter[T]) Iter[T] {
	return func() (T, bool) {
		v, ok := i1()
		if !ok {
			return i2()
		}
		return v, true
	}
}

func Chain[T any](i1, i2 Iter[T], in ...Iter[T]) Iter[T] {
	i := chain(i1, i2)
	for _, iv := range in {
		i = chain(i, iv)
	}
	return i
}
