package queue

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2/job"
	"pggat2/lib/util/rbtree"
	"pggat2/lib/util/ring"
)

type Queue struct {
	signal chan struct{}

	active uuid.UUID
	start  time.Time

	floor time.Duration

	stride    map[uuid.UUID]time.Duration
	pending   map[uuid.UUID]*ring.Ring[job.Job]
	scheduled rbtree.RBTree[time.Duration, job.Job]
	mu        sync.Mutex
}

func NewQueue() *Queue {
	return &Queue{
		stride:  make(map[uuid.UUID]time.Duration),
		pending: make(map[uuid.UUID]*ring.Ring[job.Job]),
		signal:  make(chan struct{}),
	}
}

func (T *Queue) Idle() bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	return T.active == uuid.Nil
}

func (T *Queue) HasWork() bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	_, _, ok := T.scheduled.Min()
	return ok
}

func (T *Queue) Queue(work job.Job) {
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

// schedule the next work for source
func (T *Queue) schedule(source uuid.UUID) {
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

func (T *Queue) scheduleWork(work job.Job) bool {
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

// Steal work from this Sink that is satisfied by constraints
func (T *Queue) Steal(constraints rob.Constraints) ([]job.Job, bool) {
	T.mu.Lock()
	defer T.mu.Unlock()

	for stride, work, ok := T.scheduled.Min(); ok; stride, work, ok = T.scheduled.Next(stride) {
		if constraints.Satisfies(work.Constraints) {
			// steal it
			T.scheduled.Delete(stride)

			// steal pending
			pending, _ := T.pending[work.Source]

			jobs := make([]job.Job, 0, pending.Length()+1)
			jobs = append(jobs, work)

			for work, ok = pending.PopFront(); ok; work, ok = pending.PopFront() {
				jobs = append(jobs, work)
			}

			return jobs, true
		}
	}

	// no stealable work
	return nil, false
}

func (T *Queue) Ready() <-chan struct{} {
	return T.signal
}

func (T *Queue) Read() (job.Job, bool) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.active != uuid.Nil {
		source := T.active
		dur := time.Since(T.start)
		T.active = uuid.Nil

		T.stride[source] += dur
		T.schedule(source)
	}

	stride, j, ok := T.scheduled.Min()
	if !ok {
		return job.Job{}, false
	}
	T.scheduled.Delete(stride)
	T.floor = stride

	T.active = j.Source
	T.start = time.Now()
	return j, true
}
