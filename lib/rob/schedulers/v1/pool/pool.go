package pool

import (
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v1/pool/job"
	"pggat2/lib/rob/schedulers/v1/pool/sink"
	"pggat2/lib/util/maps"
)

type Pool struct {
	affinity maps.RWLocked[uuid.UUID, uuid.UUID]

	// backlog should only be accessed when mu is Locked in some way or another
	backlog []job.Stalled
	bmu     sync.Mutex
	sinks   map[uuid.UUID]*sink.Sink
	mu      sync.RWMutex
}

func MakePool() Pool {
	return Pool{
		sinks: make(map[uuid.UUID]*sink.Sink),
	}
}

func (T *Pool) DoConcurrent(j job.Concurrent) bool {
	affinity, _ := T.affinity.Load(j.Source)

	// these can be unlocked and locked a bunch here because it is less bad if DoConcurrent misses a sink
	// (it will just stall the job and try again)
	T.mu.RLock()

	// try affinity first
	if v, ok := T.sinks[affinity]; ok {
		T.mu.RUnlock()
		if ok = v.DoConcurrent(j); ok {
			return true
		}
		T.mu.RLock()
	}

	for id, v := range T.sinks {
		if id == affinity {
			continue
		}
		T.mu.RUnlock()
		if ok := v.DoConcurrent(j); ok {
			// set affinity
			T.affinity.Store(j.Source, id)

			return true
		}
		T.mu.RLock()
	}

	T.mu.RUnlock()
	return false
}

func (T *Pool) DoStalled(j job.Stalled) {
	affinity, _ := T.affinity.Load(j.Source)

	T.mu.RLock()
	defer T.mu.RUnlock()

	// try affinity first
	if v, ok := T.sinks[affinity]; ok {
		if ok = v.DoStalled(j); ok {
			return
		}
	}

	for id, v := range T.sinks {
		if id == affinity {
			continue
		}

		if ok := v.DoStalled(j); ok {
			T.affinity.Store(j.Source, id)
			return
		}
	}

	// add to backlog
	T.bmu.Lock()
	defer T.bmu.Unlock()
	T.backlog = append(T.backlog, j)
}

func (T *Pool) AddWorker(constraints rob.Constraints, worker rob.Worker) {
	id := uuid.New()
	s := sink.NewSink(id, constraints, worker)

	T.mu.Lock()
	defer T.mu.Unlock()
	// if mu is locked, we don't need to lock bmu, because we are the only accessor
	T.sinks[id] = s
	i := 0
	for _, v := range T.backlog {
		if ok := s.DoStalled(v); !ok {
			T.backlog[i] = v
			i++
		}
	}
	T.backlog = T.backlog[:i]
}

func (T *Pool) stealFor(id uuid.UUID) {
	T.mu.RLock()

	s := T.sinks[id]

	for _, v := range T.sinks {
		if v == s {
			continue
		}

		T.mu.RUnlock()
		if src := v.StealFor(s); src != uuid.Nil {
			T.affinity.Store(src, id)
			return
		}
		T.mu.RLock()
	}

	T.mu.RUnlock()
}

func (T *Pool) Do(id uuid.UUID, constraints rob.Constraints, work any) {
	T.mu.RLock()
	s := T.sinks[id]
	T.mu.RUnlock()

	if !s.Do(constraints, work) {
		// try to steal
		T.stealFor(id)
	}
}
