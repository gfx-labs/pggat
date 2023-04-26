package iter

type Iter[T any] func() (T, bool)
