package schedulers

import (
	"github.com/google/uuid"
	"sync"
	"time"
	"tuxpa.in/a/zlog/log"

	"pggat/lib/rob"
	"pggat/lib/rob/schedulers/v2/job"
	"pggat/lib/rob/schedulers/v2/sink"
	"pggat/lib/util/maps"
	"pggat/lib/util/pools"
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

func Test(s *Scheduler) {
	go func() {
		for {
			time.Sleep(1 * time.Second)
			func() {
				s.mu.RLock()
				defer s.mu.RUnlock()

				s.bmu.Lock()
				defer s.bmu.Unlock()
				log.Printf("%d sinks | %d backlogged jobs", len(s.sinks), len(s.backlog))
			}()
		}
	}()
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
		for _, v := range T.backlog {
			s.Enqueue(v)
		}
		T.backlog = T.backlog[:0]
		return worker
	}

	T.stealFor(worker)
	return worker
}

func (T *Scheduler) DeleteWorker(worker uuid.UUID) {
	var s *sink.Sink
	var ok bool
	func() {
		T.mu.Lock()
		defer T.mu.Unlock()
		s, ok = T.sinks[worker]
		delete(T.sinks, worker)
	}()
	if !ok {
		return
	}

	// now we need to reschedule all the work that was scheduled to s (stalled only).
	jobs := s.StealAll()

	for _, j := range jobs {
		T.Enqueue(j)
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

func (T *Scheduler) TryAcquire(j job.Concurrent) uuid.UUID {
	affinity, _ := T.affinity.Load(j.User)

	T.mu.RLock()
	defer T.mu.RUnlock()

	// try affinity first
	if v, ok := T.sinks[affinity]; ok {
		if v.Acquire(j) {
			return affinity
		}
	}

	for id, v := range T.sinks {
		if id == affinity {
			continue
		}
		if v.Acquire(j) {
			// set affinity
			T.affinity.Store(j.User, id)
			return id
		}
	}

	return uuid.Nil
}

func (T *Scheduler) Enqueue(j job.Stalled) {
	affinity, _ := T.affinity.Load(j.User)

	T.mu.RLock()
	defer T.mu.RUnlock()

	// try affinity first
	if v, ok := T.sinks[affinity]; ok {
		v.Enqueue(j)
		return
	}

	for id, v := range T.sinks {
		if id == affinity {
			continue
		}

		v.Enqueue(j)
		T.affinity.Store(j.User, id)
		return
	}

	// add to backlog
	T.bmu.Lock()
	defer T.bmu.Unlock()
	T.backlog = append(T.backlog, j)
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
