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

	// metrics
	lastMetricsRead time.Time
	idle            time.Duration

	active uuid.UUID
	start  time.Time

	floor     time.Duration
	stride    map[uuid.UUID]time.Duration
	pending   map[uuid.UUID]*ring.Ring[job.Stalled]
	scheduled rbtree.RBTree[time.Duration, job.Stalled]

	mu sync.Mutex
}

func NewSink(id uuid.UUID, constraints rob.Constraints, worker rob.Worker) *Sink {
	now := time.Now()

	return &Sink{
		id:          id,
		constraints: constraints,
		worker:      worker,

		lastMetricsRead: now,
		start:           now,

		stride:  make(map[uuid.UUID]time.Duration),
		pending: make(map[uuid.UUID]*ring.Ring[job.Stalled]),
	}
}

func (T *Sink) GetWorker() rob.Worker {
	return T.worker
}

func (T *Sink) setActive(source uuid.UUID) {
	if T.active != uuid.Nil {
		panic("set active called when another was active")
	}
	now := time.Now()
	start := T.start
	if start.Before(T.lastMetricsRead) {
		start = T.lastMetricsRead
	}
	T.idle += now.Sub(start)
	T.active = source
	T.start = now
}

func (T *Sink) ExecuteConcurrent(j job.Concurrent) (ok, hasMore bool) {
	if !T.constraints.Satisfies(j.Context.Constraints) {
		return false, false
	}

	T.mu.Lock()

	if T.active != uuid.Nil {
		// this Sink is in use
		T.mu.Unlock()
		return false, false
	}

	T.setActive(j.Source)

	T.mu.Unlock()

	return true, T.Execute(j.Context, j.Work)
}

func (T *Sink) trySchedule(j job.Stalled) bool {
	if T.active == j.Source {
		// shouldn't be scheduled yet
		return false
	}

	stride, ok := T.stride[j.Source]
	if !ok {
		// set to max
		stride = T.floor
		if s, _, ok := T.scheduled.Max(); ok {
			stride = s + 1
		}
		T.stride[j.Source] = stride
	} else if stride < T.floor {
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
		return true
	}
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

func (T *Sink) ExecuteStalled(j job.Stalled) bool {
	if !T.constraints.Satisfies(j.Context.Constraints) {
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
	j, ok := pending.PopFront()
	if !ok {
		return
	}
	if ok = T.trySchedule(j); !ok {
		pending.PushFront(j)
		return
	}
}

func (T *Sink) next() bool {
	now := time.Now()
	if T.active != uuid.Nil {
		source := T.active
		dur := now.Sub(T.start)
		T.active = uuid.Nil
		T.start = now

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

func (T *Sink) Execute(ctx *rob.Context, work any) (hasMore bool) {
	T.worker.Do(ctx, work)
	if ctx.Removed {
		return false
	}

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

	for stride, j, ok := T.scheduled.Min(); ok; stride, j, ok = T.scheduled.Next(stride) {
		if rhs.constraints.Satisfies(j.Context.Constraints) {
			source := j.Source

			// take jobs from T
			T.scheduled.Delete(stride)

			pending, _ := T.pending[source]
			delete(T.pending, source)

			T.mu.Unlock()

			rhs.ExecuteStalled(j)

			for j, ok = pending.PopFront(); ok; j, ok = pending.PopFront() {
				rhs.ExecuteStalled(j)
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

func (T *Sink) StealAll() []job.Stalled {
	var all []job.Stalled

	T.mu.Lock()
	defer T.mu.Unlock()

	for {
		if k, j, ok := T.scheduled.Min(); ok {
			T.scheduled.Delete(k)
			all = append(all, j)
		} else {
			break
		}
	}

	for _, value := range T.pending {
		for {
			if j, ok := value.PopFront(); ok {
				all = append(all, j)
			} else {
				break
			}
		}
	}

	return all
}

func (T *Sink) ReadMetrics(metrics *rob.Metrics) {
	T.mu.Lock()
	defer T.mu.Unlock()

	now := time.Now()

	var lastActive time.Time

	dur := now.Sub(T.lastMetricsRead)

	if T.active == uuid.Nil {
		lastActive = T.start

		start := T.start
		if start.Before(T.lastMetricsRead) {
			start = T.lastMetricsRead
		}
		T.idle += now.Sub(start)
	}

	metrics.Workers[T.id] = rob.WorkerMetrics{
		LastActive: lastActive,

		Idle:   T.idle,
		Active: dur - T.idle,
	}

	T.lastMetricsRead = now
	T.idle = 0

	for _, pending := range T.pending {
		for i := 0; i < pending.Length(); i++ {
			j, _ := pending.Get(i)
			metrics.Jobs[j.ID] = rob.JobMetrics{
				Created: j.Created,
			}
		}
	}

	for k, v, ok := T.scheduled.Min(); ok; k, v, ok = T.scheduled.Next(k) {
		metrics.Jobs[v.ID] = rob.JobMetrics{
			Created: v.Created,
		}
	}
}
