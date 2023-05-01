package schedulers

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/util/rbtree"
)

const (
	// how often we should wake up to try and steal some work
	stealPeriod = 100 * time.Millisecond
)

type Sink struct {
	stealer     stealer
	constraints rob.Constraints

	runtime map[uuid.UUID]time.Duration

	active  *Source
	requeue bool
	start   time.Time

	queue rbtree.RBTree[time.Duration, *Source]
	floor time.Duration
	ready chan struct{}

	mu sync.Mutex
}

func newSink(stealer stealer, constraints rob.Constraints) *Sink {
	return &Sink{
		stealer:     stealer,
		constraints: constraints,
		runtime:     make(map[uuid.UUID]time.Duration),
		ready:       make(chan struct{}),
	}
}

func (T *Sink) assign(source *Source) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T._assign(source)
}

func (T *Sink) _assign(source *Source) {
	source.setNotifier(T)

	T._enqueue(source)
}

// steal a thread from this Sink. Note: you can only steal pending threads
func (T *Sink) steal() *Source {
	T.mu.Lock()
	defer T.mu.Unlock()

	rt, source, ok := T.queue.Min()
	if !ok {
		return nil
	}
	T.queue.Delete(rt)
	return source
}

func (T *Sink) notify(source *Source) {
	T.mu.Lock()
	defer T.mu.Unlock()
	T._enqueue(source)
}

func (T *Sink) _enqueue(source *Source) {
	// handle if active source
	if T.active == source {
		T.requeue = true
		return
	}

	runtime, _ := T.runtime[source.id]

	if runtime < T.floor {
		runtime = T.floor
		T.runtime[source.id] = runtime
	}

	for {
		// find unique runtime (usually will only run once)
		if v, ok := T.queue.Get(runtime); ok {
			if v == source {
				return
			}
			runtime += 1
			continue
		}

		T.queue.Set(runtime, source)
		break
	}

	select {
	case T.ready <- struct{}{}:
	default:
	}
}

func (T *Sink) _next() *Source {
	for {
		runtime, source, ok := T.queue.Min()
		if !ok {
			// unlock to allow work to be added to queue (or stolen) while we wait
			T.mu.Unlock()
			// attempt to steal
			source = T.stealer.steal(T)
			if source != nil {
				T.mu.Lock()
				T._assign(source)
			} else {
				select {
				case <-T.ready:
				case <-time.After(stealPeriod):
				}
				T.mu.Lock()
			}
			continue
		}
		T.queue.Delete(runtime)
		T.floor = runtime
		return source
	}
}

func (T *Sink) Read() any {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.active != nil {
		active := T.active
		T.active = nil

		T.runtime[active.id] += time.Since(T.start)

		if T.requeue {
			T._enqueue(active)
		}
	}

	for {
		// get next runnable thread
		T.active = T._next()

		T.start = time.Now()

		work, ok, hasMore := T.active.pop()
		if !ok {
			continue
		}

		T.requeue = hasMore

		return work
	}
}

var _ rob.Sink = (*Sink)(nil)
var _ notifier = (*Sink)(nil)
