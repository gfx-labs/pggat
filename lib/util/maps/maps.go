package maps

func FirstWhere[K comparable, V any](haystack map[K]V, predicate func(K, V) bool) (K, V, bool) {
	for k, v := range haystack {
		if predicate(k, v) {
			return k, v, true
		}
	}
	return *new(K), *new(V), false
}
