package schedulers

import (
	"github.com/google/uuid"
	"pggat/lib/rob"
	"pggat/lib/rob/schedulers/v2/job"
	"pggat/lib/rob/schedulers/v2/sink"
	"pggat/lib/util/maps"
	"pggat/lib/util/pools"
	"sync"
)

type Scheduler struct {
	affinity maps.RWLocked[uuid.UUID, uuid.UUID]

	// resource pools
	ready pools.Locked[chan uuid.UUID]

	// backlog is the list of user
	backlog []job.Stalled
	bmu     sync.Mutex
	sinks   map[uuid.UUID]*sink.Sink
	mu      sync.RWMutex
}

func (T *Scheduler) NewWorker() uuid.UUID {
	worker := uuid.New()

	s := sink.NewSink(worker)

	T.mu.Lock()
	defer T.mu.Unlock()
	// if mu is locked, we don't need to lock bmu, because we are the only accessor
	if T.sinks == nil {
		T.sinks = make(map[uuid.UUID]*sink.Sink)
	}
	T.sinks[worker] = s

	if len(T.backlog) > 0 {
		s.Enqueue(T.backlog...)
		T.backlog = T.backlog[:0]
		return worker
	}

	T.stealFor(worker)
	return worker
}

func (T *Scheduler) DeleteWorker(worker uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()
	s, ok := T.sinks[worker]
	delete(T.sinks, worker)
	if !ok {
		return
	}

	// now we need to reschedule all the work that was scheduled to s (stalled only).
	jobs := s.StealAll()

	for _, j := range jobs {
		if id := T.tryAcquire(j.Concurrent); id != uuid.Nil {
			j.Ready <- id
			continue
		}
		T.enqueue(j)
	}
}

func (*Scheduler) NewUser() uuid.UUID {
	return uuid.New()
}

func (T *Scheduler) DeleteUser(user uuid.UUID) {
	T.affinity.Delete(user)

	T.mu.RLock()
	defer T.mu.RUnlock()
	for _, v := range T.sinks {
		v.RemoveUser(user)
	}
}

func (T *Scheduler) tryAcquire(j job.Concurrent) uuid.UUID {
	affinity, _ := T.affinity.Load(j.User)

	// try affinity first
	if v, ok := T.sinks[affinity]; ok {
		if v.Acquire(j) {
			return affinity
		}
	}

	for id, v := range T.sinks {
		if v.Acquire(j) {
			// set affinity
			T.affinity.Store(j.User, id)
			return id
		}
	}

	return uuid.Nil
}

func (T *Scheduler) TryAcquire(j job.Concurrent) uuid.UUID {
	T.mu.RLock()
	defer T.mu.RUnlock()

	return T.tryAcquire(j)
}

func (T *Scheduler) enqueue(j job.Stalled) {
	affinity, _ := T.affinity.Load(j.User)

	// try affinity first
	if v, ok := T.sinks[affinity]; ok {
		v.Enqueue(j)
		return
	}

	for id, v := range T.sinks {
		v.Enqueue(j)
		T.affinity.Store(j.User, id)
		return
	}

	// add to backlog
	T.bmu.Lock()
	defer T.bmu.Unlock()
	T.backlog = append(T.backlog, j)
}

func (T *Scheduler) Enqueue(j ...job.Stalled) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	for _, jj := range j {
		T.enqueue(jj)
	}
}

func (T *Scheduler) Acquire(user uuid.UUID, mode rob.SyncMode) uuid.UUID {
	switch mode {
	case rob.SyncModeNonBlocking:
		return T.TryAcquire(job.Concurrent{
			User: user,
		})
	case rob.SyncModeBlocking:
		ready, ok := T.ready.Get()
		if !ok {
			ready = make(chan uuid.UUID, 1)
		}
		defer T.ready.Put(ready)

		j := job.Stalled{
			Concurrent: job.Concurrent{
				User: user,
			},
			Ready: ready,
		}
		T.Enqueue(j)

		return <-ready
	case rob.SyncModeTryNonBlocking:
		if id := T.Acquire(user, rob.SyncModeNonBlocking); id != uuid.Nil {
			return id
		}
		return T.Acquire(user, rob.SyncModeBlocking)
	default:
		return uuid.Nil
	}
}

func (T *Scheduler) Release(worker uuid.UUID) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	s, ok := T.sinks[worker]
	if !ok {
		return
	}
	hasMore := s.Release()
	if !hasMore {
		// try to steal
		T.stealFor(worker)
	}
}

// stealFor will try to steal work for the specified worker. Lock Scheduler.mu before executing
func (T *Scheduler) stealFor(worker uuid.UUID) {
	s, ok := T.sinks[worker]
	if !ok {
		return
	}

	for _, v := range T.sinks {
		if v == s {
			continue
		}

		if src := v.StealFor(s); src != uuid.Nil {
			T.affinity.Store(src, worker)
			return
		}
	}
}

var _ rob.Scheduler = (*Scheduler)(nil)
