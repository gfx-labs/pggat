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
	// need to keep track of constraints for work stealing
	constraints rob.Constraints
}

type Sink struct {
	active uuid.UUID
	start  time.Time

	floor time.Duration

	stride    map[uuid.UUID]time.Duration
	pending   map[uuid.UUID]*ring.Ring[job]
	scheduled rbtree.RBTree[time.Duration, job]
	signal    chan struct{}
	mu        sync.Mutex
}

func NewSink() *Sink {
	return &Sink{
		stride:  make(map[uuid.UUID]time.Duration),
		pending: make(map[uuid.UUID]*ring.Ring[job]),
		signal:  make(chan struct{}),
	}
}

func (T *Sink) Idle() bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	return T.active == uuid.Nil
}

func (T *Sink) Queue(source uuid.UUID, work any, constraints rob.Constraints) {
	T.mu.Lock()
	defer T.mu.Unlock()

	j := job{
		source:      source,
		work:        work,
		constraints: constraints,
	}

	// try to schedule right away
	if ok := T.scheduleWork(j); ok {
		return
	}

	// add to pending queue
	if _, ok := T.pending[source]; !ok {
		T.pending[source] = new(ring.Ring[job])
	}

	T.pending[source].PushBack(j)
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
	if ok = T.scheduleWork(work); !ok {
		return
	}
	pending.PopFront()
}

func (T *Sink) scheduleWork(work job) bool {
	if T.active == work.source {
		return false
	}

	stride := T.stride[work.source]
	if stride < T.floor {
		stride = T.floor
		T.stride[work.source] = stride
	}

	for {
		// find unique stride to schedule on
		if j, ok := T.scheduled.Get(stride); ok {
			if j.source == work.source {
				return false
			}
			stride += 1
			continue
		}

		T.scheduled.Set(stride, work)
		break
	}

	// signal that more work is available if someone is waiting
	select {
	case T.signal <- struct{}{}:
	default:
	}

	return true
}

func (T *Sink) findWork() {
	// Note: the sink is not locked in this func
	// TODO(garet) try to steal work
	<-T.signal
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
			T.findWork()
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
