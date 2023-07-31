package pool

import (
	"sync"
	"time"

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

func (T *Pool) ExecuteConcurrent(j job.Concurrent) bool {
	affinity, _ := T.affinity.Load(j.Source)

	// these can be unlocked and locked a bunch here because it is less bad if ExecuteConcurrent misses a sink
	// (it will just stall the job and try again)
	T.mu.RLock()

	// try affinity first
	if v, ok := T.sinks[affinity]; ok {
		T.mu.RUnlock()
		if done, hasMore := v.ExecuteConcurrent(j); done {
			if j.Context.Removed {
				T.RemoveWorker(affinity)
				return true
			}
			if !hasMore {
				T.stealFor(affinity)
			}
			return true
		}
		T.mu.RLock()
	}

	for id, v := range T.sinks {
		if id == affinity {
			continue
		}
		T.mu.RUnlock()
		if ok, hasMore := v.ExecuteConcurrent(j); ok {
			if j.Context.Removed {
				T.RemoveWorker(id)
				return true
			}

			// set affinity
			T.affinity.Store(j.Source, id)

			if !hasMore {
				T.stealFor(id)
			}

			return true
		}
		T.mu.RLock()
	}

	T.mu.RUnlock()
	return false
}

func (T *Pool) ExecuteStalled(j job.Stalled) {
	affinity, _ := T.affinity.Load(j.Source)

	T.mu.RLock()
	defer T.mu.RUnlock()

	// try affinity first
	if v, ok := T.sinks[affinity]; ok {
		if ok = v.ExecuteStalled(j); ok {
			return
		}
	}

	for id, v := range T.sinks {
		if id == affinity {
			continue
		}

		if ok := v.ExecuteStalled(j); ok {
			T.affinity.Store(j.Source, id)
			return
		}
	}

	// add to backlog
	T.bmu.Lock()
	defer T.bmu.Unlock()
	T.backlog = append(T.backlog, j)
}

func (T *Pool) AddWorker(constraints rob.Constraints, worker rob.Worker) uuid.UUID {
	id := uuid.New()
	s := sink.NewSink(id, constraints, worker)

	T.mu.Lock()
	defer T.mu.Unlock()
	// if mu is locked, we don't need to lock bmu, because we are the only accessor
	T.sinks[id] = s
	i := 0
	for _, v := range T.backlog {
		if ok := s.ExecuteStalled(v); !ok {
			T.backlog[i] = v
			i++
		}
	}
	T.backlog = T.backlog[:i]

	return id
}

func (T *Pool) GetWorker(id uuid.UUID) rob.Worker {
	T.mu.RLock()
	defer T.mu.RUnlock()
	s, ok := T.sinks[id]
	if !ok {
		return nil
	}
	return s.GetWorker()
}

func (T *Pool) GetIdleWorker() (id uuid.UUID, idleStart time.Time) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	for i, s := range T.sinks {
		start := s.IdleStart()
		if idleStart == (time.Time{}) || start.Before(idleStart) {
			idleStart = start
			id = i
		}
	}

	return
}

func (T *Pool) RemoveWorker(id uuid.UUID) rob.Worker {
	T.mu.Lock()
	s, ok := T.sinks[id]
	if !ok {
		T.mu.Unlock()
		return nil
	}
	delete(T.sinks, id)
	T.mu.Unlock()

	// now we need to reschedule all the work that was scheduled to s (stalled only).
	jobs := s.StealAll()

	for _, j := range jobs {
		T.ExecuteStalled(j)
	}

	return s.GetWorker()
}

func (T *Pool) WorkerCount() int {
	T.mu.RLock()
	defer T.mu.RUnlock()
	return len(T.sinks)
}

func (T *Pool) stealFor(id uuid.UUID) {
	T.mu.RLock()

	s, ok := T.sinks[id]
	if !ok {
		T.mu.RUnlock()
		return
	}

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

func (T *Pool) Execute(id uuid.UUID, ctx *rob.Context, work any) {
	T.mu.RLock()
	s := T.sinks[id]
	T.mu.RUnlock()

	hasMore := s.Execute(ctx, work)
	if ctx.Removed {
		// remove
		T.RemoveWorker(id)
		return
	}
	if !hasMore {
		// try to steal
		T.stealFor(id)
	}
}

func (T *Pool) ReadMetrics(metrics *rob.Metrics) {
	maps.Clear(metrics.Jobs)
	if metrics.Jobs == nil {
		metrics.Jobs = make(map[uuid.UUID]rob.JobMetrics)
	}
	maps.Clear(metrics.Workers)
	if metrics.Workers == nil {
		metrics.Workers = make(map[uuid.UUID]rob.WorkerMetrics)
	}

	T.mu.RLock()
	defer T.mu.RUnlock()

	T.bmu.Lock()
	for _, j := range T.backlog {
		metrics.Jobs[j.ID] = rob.JobMetrics{
			Created:    j.Created,
			Backlogged: true,
		}
	}
	T.bmu.Unlock()

	for _, worker := range T.sinks {
		worker.ReadMetrics(metrics)
	}
}
