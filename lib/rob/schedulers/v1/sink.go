package schedulers

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/util/rbtree"
)

type Sink struct {
	runtime map[uuid.UUID]time.Duration

	active  *Source
	requeue bool
	start   time.Time

	queue rbtree.RBTree[time.Duration, *Source]
	floor time.Duration
	ready chan struct{}

	mu sync.Mutex
}

func newSink() *Sink {
	return &Sink{
		runtime: make(map[uuid.UUID]time.Duration),
		ready:   make(chan struct{}),
	}
}

func (T *Sink) assign(source *Source) {
	source.setNotifier(T.enqueue)

	T.enqueue(source)
}

func (T *Sink) enqueue(source *Source) {
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
		for {
			var runtime time.Duration
			var ok bool
			runtime, T.active, ok = T.queue.Min()
			if !ok {
				T.mu.Unlock()
				<-T.ready
				T.mu.Lock()
				continue
			}
			T.queue.Delete(runtime)
			T.floor = runtime
			break
		}

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
