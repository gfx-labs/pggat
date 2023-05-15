package pool

import (
	"github.com/google/uuid"
	"pggat2/lib/rob/schedulers/v0/job"
	"pggat2/lib/rob/schedulers/v0/sink"
	"sync"
)

type Pool struct {
	sinks map[uuid.UUID]*sink.Sink
	mu    sync.RWMutex
}

// Do attempts to do the work.
// Returns true if the work was done, otherwise the sender should wait on their stall chan for the next node
func (T *Pool) Do(j job.Job) bool {
	T.mu.RLock()
	defer T.mu.RUnlock()

	// TODO(garet) choose affinity, prefer idle nodes
	for _, v := range T.sinks {
		if v.DoIfIdle(j) {
			return
		}
	}

	panic("no available sinks")
}

func (T *Pool) AddSink(s *sink.Sink) {
	T.mu.Lock()
	defer T.mu.Unlock()
	T.sinks[s.ID()] = s
}
