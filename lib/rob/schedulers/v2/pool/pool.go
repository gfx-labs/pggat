package pool

import (
	"math/rand"
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2/job"
	"pggat2/lib/rob/schedulers/v2/sink"
)

type Pool struct {
	affinity  map[uuid.UUID]int
	sinks     []*sink.Sink
	backorder []job.Job
	mu        sync.Mutex
}

func MakePool() Pool {
	return Pool{
		affinity: make(map[uuid.UUID]int),
	}
}

func (T *Pool) NewSink(constraints rob.Constraints) *sink.Sink {
	snk := sink.NewSink(constraints)

	T.mu.Lock()
	defer T.mu.Unlock()

	T.sinks = append(T.sinks, snk)

	i := 0
	for _, j := range T.backorder {
		if constraints.Satisfies(j.Constraints) {
			snk.Queue(j)
		} else {
			T.backorder[i] = j
			i++
		}
	}
	T.backorder = T.backorder[:i]

	return snk
}

func (T *Pool) Schedule(work job.Job) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if len(T.sinks) == 0 {
		T.backorder = append(T.backorder, work)
		return
	}

	affinity, ok := T.affinity[work.Source]
	if !ok {
		affinity = rand.Intn(len(T.sinks))
		T.affinity[work.Source] = affinity
	}

	snk := T.sinks[affinity]
	if !snk.Constraints().Satisfies(work.Constraints) || !snk.Idle() {
		// choose a new affinity that satisfies constraints
		ok = false
		for id, s := range T.sinks {
			if s.Constraints().Satisfies(work.Constraints) {
				current := id == affinity
				snk = s
				affinity = id
				ok = true
				if !current && s.Idle() {
					// prefer idle core, if not idle try to see if we can find one that is
					break
				}
			}
		}
		if !ok {
			T.backorder = append(T.backorder, work)
			return
		}
		T.affinity[work.Source] = affinity
	}

	// yay, queued
	snk.Queue(work)
}
