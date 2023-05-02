package sink

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2/job"
	"pggat2/lib/util/rbtree"
	"pggat2/lib/util/ring"
)

type Sink struct {
	constraints rob.Constraints

	active uuid.UUID
	start  time.Time

	floor time.Duration

	stride    map[uuid.UUID]time.Duration
	pending   map[uuid.UUID]*ring.Ring[job.Job]
	scheduled rbtree.RBTree[time.Duration, job.Job]
	signal    chan struct{}
	mu        sync.Mutex
}

func NewSink(constraints rob.Constraints) *Sink {
	return &Sink{
		constraints: constraints,
		stride:      make(map[uuid.UUID]time.Duration),
		pending:     make(map[uuid.UUID]*ring.Ring[job.Job]),
		signal:      make(chan struct{}),
	}
}

func (T *Sink) Constraints() rob.Constraints {
	// no lock needed because these never change
	return T.constraints
}

func (T *Sink) Idle() bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	return T.active == uuid.Nil
}

func (T *Sink) Queue(work job.Job) {
	T.mu.Lock()
	defer T.mu.Unlock()

	// try to schedule right away
	if ok := T.scheduleWork(work); ok {
		return
	}

	// add to pending queue
	if _, ok := T.pending[work.Source]; !ok {
		T.pending[work.Source] = new(ring.Ring[job.Job])
	}

	T.pending[work.Source].PushBack(work)
}

func (T *Sink) Steal(constraints rob.Constraints) (job.Job, *ring.Ring[job.Job], bool) {
	T.mu.Lock()
	defer T.mu.Unlock()

	iter := T.scheduled.Iter()
	for stride, work, ok := iter(); ok; stride, work, ok = iter() {
		if constraints.Satisfies(work.Constraints) {
			// steal it
			T.scheduled.Delete(stride)

			// steal pending
			pending, _ := T.pending[work.Source]
			if pending.Length() == 0 {
				pending = nil
			} else {
				delete(T.pending, work.Source)
			}

			return work, pending, true
		}
	}

	// no stealable work
	return job.Job{}, nil, false
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

func (T *Sink) scheduleWork(work job.Job) bool {
	if T.active == work.Source {
		return false
	}

	stride := T.stride[work.Source]
	if stride < T.floor {
		stride = T.floor
		T.stride[work.Source] = stride
	}

	for {
		// find unique stride to schedule on
		if j, ok := T.scheduled.Get(stride); ok {
			if j.Source == work.Source {
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

		T.active = j.Source
		T.start = time.Now()
		return j.Work
	}
}

var _ rob.Sink = (*Sink)(nil)
