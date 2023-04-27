package v0

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
)

const (
	// tIdleTimeout is how long a thread can be idle before it is no longer runnable
	tIdleTimeout = 5 * time.Millisecond
)

type Sink struct {
	scheduler *Scheduler

	threads map[uuid.UUID]*thread

	active      *thread
	activeStart time.Time

	sigRunnable chan struct{}
	// TODO(garet) change for red black tree
	runnable []*thread

	minRuntime time.Duration

	mu sync.Mutex
}

func newSink(scheduler *Scheduler) *Sink {
	sink := &Sink{
		scheduler: scheduler,

		threads: make(map[uuid.UUID]*thread),

		sigRunnable: make(chan struct{}),
	}
	return sink
}

func (T *Sink) newThread(source *Source) *thread {
	t := &thread{
		source: source,
	}
	T.threads[source.id] = t
	return t
}

func (T *Sink) getOrCreateThread(source *Source) *thread {
	var t *thread
	var ok bool
	// get or create thread
	if t, ok = T.threads[source.id]; !ok {
		t = T.newThread(source)
	}
	return t
}

func (T *Sink) enqueueRunnable(t *thread) {
	if len(t.queue) == 0 {
		panic("tried to enqueue a stalled thread")
	}

	if t == T.active {
		return
	}

	T.runnable = append(T.runnable, t)
	sort.Slice(
		T.runnable,
		func(i, j int) bool {
			return T.runnable[i].runtime > T.runnable[j].runtime
		},
	)

	select {
	case T.sigRunnable <- struct{}{}:
	default:
	}
}

func (T *Sink) popThread() *thread {
	t := T.runnable[len(T.runnable)-1]
	T.runnable = T.runnable[:len(T.runnable)-1]
	return t
}

func (T *Sink) Read() any {
	T.mu.Lock()
	defer T.mu.Unlock()

	now := time.Now()

	if T.active != nil {
		t := T.active
		T.active = nil

		t.finish = now
		dur := now.Sub(T.activeStart)
		t.runtime += dur

		// reschedule if thread has more work
		if len(t.queue) > 0 {
			T.enqueueRunnable(t)
		}
	}

	for len(T.runnable) == 0 {
		T.mu.Unlock()
		<-T.sigRunnable
		T.mu.Lock()
	}

	// pop thread off
	t := T.popThread()

	T.minRuntime = t.runtime
	T.active = t
	T.activeStart = now

	// pop work from thread
	w := t.popWork()

	return w.payload
}

func (T *Sink) enqueue(w *work) {
	T.mu.Lock()
	defer T.mu.Unlock()

	t := T.getOrCreateThread(w.source)

	if len(t.queue) == 0 && time.Since(t.finish) > tIdleTimeout {
		t.runtime = T.minRuntime
	}

	t.enqueue(w)
	T.enqueueRunnable(t)
}

var _ rob.Sink = (*Sink)(nil)
