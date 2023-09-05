package maps

func Clear[K comparable, V any](m map[K]V) {
	for k := range m {
		delete(m, k)
	}
}
