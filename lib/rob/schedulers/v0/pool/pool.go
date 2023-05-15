package pool

import (
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/rob/schedulers/v0/job"
	"pggat2/lib/rob/schedulers/v0/sink"
	"pggat2/lib/util/maps"
)

type Pool struct {
	affinity maps.RWLocked[uuid.UUID, uuid.UUID]

	backlog []job.Stalled
	bmu     sync.Mutex

	sinks maps.RWLocked[uuid.UUID, *sink.Sink]
}

func MakePool() Pool {
	return Pool{}
}

// DoConcurrent attempts to do the work now.
// Returns true if the work was done, otherwise the sender should stall the work.
func (T *Pool) DoConcurrent(j job.Concurrent) (done bool) {
	affinity, _ := T.affinity.Load(j.Source)

	// try affinity first
	if v, ok := T.sinks.Load(affinity); ok {
		if done = v.DoConcurrent(j); done {
			return
		}
	}

	T.sinks.Range(func(id uuid.UUID, v *sink.Sink) bool {
		if id == affinity {
			return true
		}
		if done = v.DoConcurrent(j); done {
			// set affinity
			T.affinity.Store(j.Source, id)
			return false
		}
		return true
	})
	if done {
		return
	}

	return false
}

// DoStalled queues a job to be done eventually
func (T *Pool) DoStalled(j job.Stalled) {
	affinity, _ := T.affinity.Load(j.Source)

	// try affinity first
	if v, ok := T.sinks.Load(affinity); ok {
		if ok = v.DoStalled(j); ok {
			return
		}
	}

	var ok bool
	T.sinks.Range(func(id uuid.UUID, v *sink.Sink) bool {
		if id == affinity {
			return true
		}
		if ok = v.DoStalled(j); ok {
			// set affinity
			T.affinity.Store(j.Source, id)
			return false
		}
		return true
	})
	if ok {
		return
	}

	// add to backlog
	T.bmu.Lock()
	defer T.bmu.Unlock()
	T.backlog = append(T.backlog, j)
}

func (T *Pool) AddSink(s *sink.Sink) {
	T.sinks.Store(s.ID(), s)

	T.bmu.Lock()
	defer T.bmu.Unlock()
	i := 0
	for _, v := range T.backlog {
		if ok := s.DoStalled(v); !ok {
			T.backlog[i] = v
			i++
		}
	}
	T.backlog = T.backlog[:i]
}
