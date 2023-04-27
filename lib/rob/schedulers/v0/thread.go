package v0

import "time"

type thread struct {
	source *Source
	// runtime of thread (similar to vruntime in CFS)
	runtime time.Duration

	finish time.Time

	queue []*work
}

func (T *thread) popWork() *work {
	w := T.queue[0]
	for i := 1; i < len(T.queue)-1; i++ {
		T.queue[i] = T.queue[i+1]
	}
	T.queue = T.queue[:len(T.queue)-1]
	return w
}

func (T *thread) enqueue(w *work) {
	T.queue = append(T.queue, w)
}
