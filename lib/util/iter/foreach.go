package iter

// ForEach is the same as doing `for v, ok := iter(); ok; v, ok = iter() {...}` but cleaner on the eyes.
// If you need to break out early, use the `for` form
func ForEach[T any](iter Iter[T], f func(T)) {
	for v, ok := iter(); ok; v, ok = iter() {
		f(v)
	}
}
