package maps

import "sync"

type RWLocked[K comparable, V any] struct {
	inner map[K]V
	mu    sync.RWMutex
}

func (T *RWLocked[K, V]) Delete(key K) {
	T.mu.Lock()
	defer T.mu.Unlock()
	delete(T.inner, key)
}

func (T *RWLocked[K, V]) Load(key K) (value V, ok bool) {
	T.mu.RLock()
	defer T.mu.RUnlock()
	value, ok = T.inner[key]
	return
}

func (T *RWLocked[K, V]) LoadAndDelete(key K) (value V, loaded bool) {
	T.mu.Lock()
	defer T.mu.Unlock()
	value, loaded = T.inner[key]
	delete(T.inner, key)
	return
}

func (T *RWLocked[K, V]) LoadOrStore(key K, value V) (actual V, loaded bool) {
	T.mu.Lock()
	defer T.mu.Unlock()
	actual, loaded = T.inner[key]
	if !loaded {
		if T.inner == nil {
			T.inner = make(map[K]V)
		}
		T.inner[key] = value
		actual = value
	}
	return
}

func (T *RWLocked[K, V]) Store(key K, value V) {
	T.mu.Lock()
	defer T.mu.Unlock()
	if T.inner == nil {
		T.inner = make(map[K]V)
	}
	T.inner[key] = value
}

func (T *RWLocked[K, V]) Swap(key K, value V) (previous V, loaded bool) {
	T.mu.Lock()
	defer T.mu.Unlock()
	previous, loaded = T.inner[key]
	if T.inner == nil {
		T.inner = make(map[K]V)
	}
	T.inner[key] = value
	return
}

func (T *RWLocked[K, V]) Range(fn func(key K, value V) bool) bool {
	// this is ok because if fn panics the map will be unlocked
	T.mu.RLock()
	for k, v := range T.inner {
		T.mu.RUnlock()
		if !fn(k, v) {
			return false
		}
		T.mu.RLock()
	}
	T.mu.RUnlock()
	return true
}
