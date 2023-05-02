package sink

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/util/rbtree"
	"pggat2/lib/util/ring"
)

type job struct {
	source uuid.UUID
	work   any
}

type Sink struct {
	active uuid.UUID
	start  time.Time

	floor time.Duration

	stride    map[uuid.UUID]time.Duration
	pending   map[uuid.UUID]*ring.Ring[any]
	scheduled rbtree.RBTree[time.Duration, job]
	mu        sync.Mutex
}

func NewSink() *Sink {
	return &Sink{
		stride:  make(map[uuid.UUID]time.Duration),
		pending: make(map[uuid.UUID]*ring.Ring[any]),
	}
}

func (T *Sink) Queue(source uuid.UUID, work any) {
	T.mu.Lock()
	defer T.mu.Unlock()

	// try to schedule right away
	if ok := T.scheduleWork(source, work); ok {
		return
	}

	// add to pending queue
	if _, ok := T.pending[source]; !ok {
		T.pending[source] = new(ring.Ring[any])
	}

	T.pending[source].PushBack(work)
}

// schedule the next work for source
func (T *Sink) schedule(source uuid.UUID) {
	pending, ok := T.pending[source]
	if !ok {
		return
	}
	work, ok := pending.Get(0)
	if !ok {
		return
	}
	if ok = T.scheduleWork(source, work); !ok {
		return
	}
	pending.PopFront()
}

func (T *Sink) scheduleWork(source uuid.UUID, work any) bool {
	if T.active == source {
		return false
	}

	stride := T.stride[source]
	if stride < T.floor {
		stride = T.floor
		T.stride[source] = stride
	}

	for {
		// find unique stride to schedule on
		if j, ok := T.scheduled.Get(stride); ok {
			if j.source == source {
				return false
			}
			stride += 1
			continue
		}

		T.scheduled.Set(stride, job{
			source: source,
			work:   work,
		})
		break
	}

	return true
}

func (T *Sink) Read() any {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.active != uuid.Nil {
		source := T.active
		dur := time.Since(T.start)
		T.active = uuid.Nil

		T.stride[source] += dur
		T.schedule(source)
	}

	for {
		stride, j, ok := T.scheduled.Min()
		if !ok {
			T.mu.Unlock()
			// TODO(garet) try to steal or sleep until more work is available
			T.mu.Lock()
			continue
		}
		T.scheduled.Delete(stride)
		T.floor = stride

		T.active = j.source
		T.start = time.Now()
		return j.work
	}
}

var _ rob.Sink = (*Sink)(nil)
