package maps

type twoKeys[K1, K2 comparable] struct {
	first  K1
	second K2
}

type TwoKey[K1, K2 comparable, V any] struct {
	inner map[twoKeys[K1, K2]]V
}

func (T *TwoKey[K1, K2, V]) Delete(k1 K1, k2 K2) {
	delete(T.inner, twoKeys[K1, K2]{
		first:  k1,
		second: k2,
	})
}

func (T *TwoKey[K1, K2, V]) Load(k1 K1, k2 K2) (value V, ok bool) {
	value, ok = T.inner[twoKeys[K1, K2]{
		first:  k1,
		second: k2,
	}]
	return
}

func (T *TwoKey[K1, K2, V]) Store(k1 K1, k2 K2, value V) {
	if T.inner == nil {
		T.inner = make(map[twoKeys[K1, K2]]V)
	}
	T.inner[twoKeys[K1, K2]{
		first:  k1,
		second: k2,
	}] = value
}

func (T *TwoKey[K1, K2, V]) Range(fn func(K1, K2, V) bool) bool {
	for keys, value := range T.inner {
		if !fn(keys.first, keys.second, value) {
			return false
		}
	}
	return true
}
