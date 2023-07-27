package chans

func TrySend[T any](ch chan<- T, value T) bool {
	select {
	case ch <- value:
		return true
	default:
		return false
	}
}

func TryRecv[T any](ch <-chan T) (T, bool) {
	select {
	case value, ok := <-ch:
		return value, ok
	default:
		return *new(T), false
	}
}
