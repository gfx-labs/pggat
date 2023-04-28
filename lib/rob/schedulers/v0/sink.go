package v0

import (
	"sort"
	"sync"
	"time"

	"pggat2/lib/rob"
)

type Sink struct {
	// currently active Source
	active *Source
	// start time of the current thread
	start time.Time

	awake chan struct{}

	// TODO(garet) change for red black tree
	// runnable queue
	queue []*Source

	mu sync.Mutex
}

func newSink() *Sink {
	sink := &Sink{
		awake: make(chan struct{}),
	}
	return sink
}

func (T *Sink) _runnable(t *Source) {
	for _, q := range T.queue {
		if t == q {
			return
		}
	}

	T.queue = append(T.queue, t)
	sort.Slice(
		T.queue,
		func(i, j int) bool {
			return T.queue[i].runtime.Load() > T.queue[j].runtime.Load()
		},
	)

	select {
	case T.awake <- struct{}{}:
	default:
	}
}

func (T *Sink) runnable(t *Source) {
	if t.idle() {
		panic("tried to enqueue a stalled thread")
	}

	T.mu.Lock()
	defer T.mu.Unlock()

	if T.active == t {
		return
	}

	T._runnable(t)
}

func (T *Sink) _next() *Source {
	t := T.queue[len(T.queue)-1]
	T.queue = T.queue[:len(T.queue)-1]
	return t
}

func (T *Sink) Read() any {
	T.mu.Lock()
	defer T.mu.Unlock()

	now := time.Now()

	if T.active != nil {
		t := T.active
		T.active = nil

		dur := now.Sub(T.start)
		t.runtime.Add(dur.Nanoseconds())

		// reschedule if thread has more work
		if !t.idle() {
			T._runnable(t)
		}
	}

	for len(T.queue) == 0 {
		T.mu.Unlock()
		<-T.awake
		T.mu.Lock()
	}

	// pop thread off
	t := T._next()

	T.active = t
	T.start = now

	// pop work from thread
	return t.pop()
}

var _ rob.Sink = (*Sink)(nil)
