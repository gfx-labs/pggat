package maps

func Clone[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}

	m2 := make(map[K]V, len(m))
	for k, v := range m {
		m2[k] = v
	}

	return m2
}
