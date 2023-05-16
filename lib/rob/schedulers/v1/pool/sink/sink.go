package sink

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v1/pool/job"
	"pggat2/lib/util/rbtree"
	"pggat2/lib/util/ring"
)

type Sink struct {
	id          uuid.UUID
	constraints rob.Constraints
	worker      rob.Worker

	// non final

	active uuid.UUID
	start  time.Time

	floor     time.Duration
	stride    map[uuid.UUID]time.Duration
	pending   map[uuid.UUID]*ring.Ring[job.Stalled]
	scheduled rbtree.RBTree[time.Duration, job.Stalled]

	mu sync.Mutex
}

func NewSink(id uuid.UUID, constraints rob.Constraints, worker rob.Worker) *Sink {
	return &Sink{
		id:          id,
		constraints: constraints,
		worker:      worker,

		stride:  make(map[uuid.UUID]time.Duration),
		pending: make(map[uuid.UUID]*ring.Ring[job.Stalled]),
	}
}

func (T *Sink) setActive(source uuid.UUID) {
	if T.active != uuid.Nil {
		panic("set active called when another was active")
	}
	T.active = source
	T.start = time.Now()
}

func (T *Sink) DoConcurrent(j job.Concurrent) bool {
	if !T.constraints.Satisfies(j.Constraints) {
		return false
	}

	T.mu.Lock()

	if T.active != uuid.Nil {
		// this Sink is in use
		T.mu.Unlock()
		return false
	}

	T.setActive(j.Source)

	T.mu.Unlock()

	T.Do(j.Constraints, j.Work)
	return true
}

func (T *Sink) trySchedule(j job.Stalled) bool {
	if T.active == j.Source {
		// shouldn't be scheduled yet
		return false
	}

	stride := T.stride[j.Source]
	if stride < T.floor {
		stride = T.floor
		T.stride[j.Source] = stride
	}

	for {
		// find unique stride to schedule on
		if s, ok := T.scheduled.Get(stride); ok {
			if s.Source == j.Source {
				return false
			}
			stride += 1
			continue
		}

		T.scheduled.Set(stride, j)
		break
	}

	return true
}

func (T *Sink) enqueue(j job.Stalled) {
	if T.trySchedule(j) {
		return
	}

	p, ok := T.pending[j.Source]

	// add to pending queue
	if !ok {
		p = ring.NewRing[job.Stalled](0, 1)
		T.pending[j.Source] = p
	}

	p.PushBack(j)
}

func (T *Sink) DoStalled(j job.Stalled) bool {
	if !T.constraints.Satisfies(j.Constraints) {
		return false
	}

	// enqueue job
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.active == uuid.Nil {
		// run it now
		T.setActive(j.Source)
		j.Ready <- T.id
		return true
	}

	// enqueue for running later
	T.enqueue(j)
	return true
}

func (T *Sink) enqueueNextFor(source uuid.UUID) {
	pending, ok := T.pending[source]
	if !ok {
		return
	}
	j, ok := pending.Get(0)
	if !ok {
		return
	}
	if ok = T.trySchedule(j); !ok {
		return
	}
	pending.PopFront()
}

func (T *Sink) next() bool {
	if T.active != uuid.Nil {
		source := T.active
		dur := time.Since(T.start)
		T.active = uuid.Nil

		T.stride[source] += dur

		T.enqueueNextFor(source)
	}

	stride, j, ok := T.scheduled.Min()
	if !ok {
		return false
	}
	T.scheduled.Delete(stride)
	if stride > T.floor {
		T.floor = stride
	}

	T.setActive(j.Source)
	j.Ready <- T.id
	return true
}

func (T *Sink) Do(constraints rob.Constraints, work any) (hasMore bool) {
	T.worker.Do(constraints, work)

	// queue next
	T.mu.Lock()
	defer T.mu.Unlock()
	return T.next()
}

func (T *Sink) StealFor(rhs *Sink) uuid.UUID {
	if T == rhs {
		// cannot steal from ourselves
		return uuid.Nil
	}

	T.mu.Lock()
	defer T.mu.Unlock()

	for stride, work, ok := T.scheduled.Min(); ok; stride, work, ok = T.scheduled.Next(stride) {
		if rhs.constraints.Satisfies(work.Constraints) {
			source := work.Source

			// take jobs from T
			T.scheduled.Delete(stride)

			pending, _ := T.pending[source]
			delete(T.pending, source)

			T.mu.Unlock()

			rhs.DoStalled(work)

			for work, ok = pending.PopFront(); ok; work, ok = pending.PopFront() {
				rhs.DoStalled(work)
			}

			T.mu.Lock()

			if pending != nil {
				if _, ok = T.pending[source]; !ok {
					T.pending[source] = pending
				}
			}

			return source
		}
	}

	return uuid.Nil
}
