package pools

import (
	"math/bits"

	"pggat/lib/util/slices"
)

type Log2[T any] struct {
	pools [32]Pool[[]T]
}

func (L *Log2[T]) Get(length int32) []T {
	if length == 0 {
		return nil
	}

	log2 := bits.Len32(uint32(length - 1))
	v, ok := L.pools[log2].Get()
	if ok {
		v = slices.Resize(v, int(length))
		slices.Clear(v)
		return v
	}
	capacity := 1 << log2
	v = make([]T, capacity)
	return v[:length]
}

func (L *Log2[T]) Put(v []T) {
	if cap(v) == 0 {
		return
	}
	log2 := bits.TrailingZeros32(uint32(cap(v)))
	L.pools[log2].Put(v)
}
