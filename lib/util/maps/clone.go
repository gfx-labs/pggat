package maps

func Clone[K comparable, V any](value map[K]V) map[K]V {
	if value == nil {
		return nil
	}
	m := make(map[K]V, len(value))
	for k, v := range value {
		m[k] = v
	}
	return m
}
