package sink

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v0/job"
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

	floor time.Duration

	stride    map[uuid.UUID]time.Duration
	pending   map[uuid.UUID]*ring.Ring[job.Stalled]
	scheduled rbtree.RBTree[time.Duration, job.Stalled]

	mu sync.Mutex
}

func NewSink(constraints rob.Constraints, worker rob.Worker) *Sink {
	return &Sink{
		id:          uuid.New(),
		constraints: constraints,
		worker:      worker,

		stride:  make(map[uuid.UUID]time.Duration),
		pending: make(map[uuid.UUID]*ring.Ring[job.Stalled]),
	}
}

func (T *Sink) ID() uuid.UUID {
	return T.id
}

func (T *Sink) setNext(source uuid.UUID) {
	T.active = source
	T.start = time.Now()
}

func (T *Sink) addStalled(j job.Stalled) {
	if T.active != uuid.Nil {
		// sink is in use, add to queue
		T.scheduleStalled(j)
	} else {
		// sink is open, do now
		T.setNext(j.Source)
		j.Out <- T
	}
}

func (T *Sink) scheduleStalled(j job.Stalled) {
	// try to schedule right away
	if ok := T.tryScheduleStalled(j); ok {
		return
	}

	// add to pending queue
	if _, ok := T.pending[j.Source]; !ok {
		r := ring.NewRing[job.Stalled](0, 1)
		r.PushBack(j)
		T.pending[j.Source] = r
		return
	}

	T.pending[j.Source].PushBack(j)
}

func (T *Sink) schedulePending(source uuid.UUID) {
	pending, ok := T.pending[source]
	if !ok {
		return
	}
	work, ok := pending.Get(0)
	if !ok {
		return
	}
	if ok = T.tryScheduleStalled(work); !ok {
		return
	}
	pending.PopFront()
}

func (T *Sink) tryScheduleStalled(j job.Stalled) bool {
	if T.active == j.Source {
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

func (T *Sink) next() bool {
	if T.active != uuid.Nil {
		source := T.active
		dur := time.Since(T.start)
		T.active = uuid.Nil

		T.stride[source] += dur

		T.schedulePending(source)
	}

	stride, j, ok := T.scheduled.Min()
	if !ok {
		return false
	}
	T.scheduled.Delete(stride)
	T.floor = stride

	T.setNext(j.Source)
	j.Out <- T
	return true
}

func (T *Sink) DoConcurrent(j job.Concurrent) (done, hasMore bool) {
	if !T.constraints.Satisfies(j.Constraints) {
		return false, true
	}

	T.mu.Lock()

	if T.active != uuid.Nil {
		T.mu.Unlock()
		// this Sink is in use
		return false, true
	}

	T.setNext(j.Source)
	T.mu.Unlock()

	return true, T.Do(j.Constraints, j.Work)
}

func (T *Sink) DoStalled(j job.Stalled) (ok bool) {
	if !T.constraints.Satisfies(j.Constraints) {
		return false
	}

	T.mu.Lock()
	defer T.mu.Unlock()

	T.addStalled(j)

	return true
}

func (T *Sink) Do(constraints rob.Constraints, work any) bool {
	if !T.constraints.Satisfies(constraints) {
		panic("Do called on sink with non satisfied constraints")
	}
	T.worker.Do(constraints, work)
	T.mu.Lock()
	defer T.mu.Unlock()
	return T.next()
}

func (T *Sink) StealFor(rhs *Sink) (uuid.UUID, bool) {
	if T == rhs {
		// cannot steal from ourselves
		return uuid.Nil, false
	}

	T.mu.Lock()
	defer T.mu.Unlock()

	for stride, work, ok := T.scheduled.Min(); ok; stride, work, ok = T.scheduled.Next(stride) {
		if rhs.constraints.Satisfies(work.Constraints) {
			source := work.Source

			// take jobs from T
			T.scheduled.Delete(stride)

			pending, _ := T.pending[work.Source]
			delete(T.pending, work.Source)

			// we have to unlock to prevent deadlock
			// if T tries to steal from rhs at the same time rhs tries to steal from T, we could deadlock otherwise
			// (speaking from experience)
			T.mu.Unlock()

			// add to rhs
			rhs.mu.Lock()

			rhs.addStalled(work)

			for work, ok = pending.PopFront(); ok; work, ok = pending.PopFront() {
				rhs.addStalled(work)
			}

			rhs.mu.Unlock()

			// try to return buffer to T (if fails, it's not a big deal)

			T.mu.Lock()

			if _, ok = T.pending[work.Source]; !ok {
				T.pending[work.Source] = pending
			}

			return source, true
		}
	}

	return uuid.Nil, false
}
