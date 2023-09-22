package slices

// Sorted is a sorted slice. As long as all items are inserted by Insert, updated by Update, and removed by Delete,
// this slice will stay sorted
type Sorted[V any] []V

func (T Sorted[V]) Insert(value V, sorter func(V) int) Sorted[V] {
	key := sorter(value)
	for i, v := range T {
		if sorter(v) < key {
			continue
		}

		res := append(T, *new(V))
		copy(res[i+1:], res[i:])
		res[i] = value
		return res
	}

	return append(T, value)
}

func (T Sorted[V]) Update(index int, sorter func(V) int) {
	value := T[index]
	key := sorter(value)

	for i, v := range T {
		switch {
		case i < index:
			if sorter(v) < key {
				continue
			}

			// move all up by one, move from index to i
			copy(T[i+1:], T[i:index])
			T[i] = value
			return
		case i > index:
			if sorter(v) < key {
				continue
			}

			// move all down by one, move from index to i
			copy(T[index:], T[index+1:i])
			T[i-1] = value
			return
		default:
			continue
		}
	}

	// move all down by one, move from index to i
	copy(T[index:], T[index+1:])
	T[len(T)-1] = value
}
