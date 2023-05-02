package pool

import (
	"math/rand"
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2/sink"
)

type sinkAndConstraints struct {
	sink        *sink.Sink
	constraints rob.Constraints
}

type job struct {
	source      uuid.UUID
	work        any
	constraints rob.Constraints
}

type Pool struct {
	affinity  map[uuid.UUID]int
	sinks     []sinkAndConstraints
	backorder []job
	mu        sync.Mutex
}

func MakePool() Pool {
	return Pool{
		affinity: make(map[uuid.UUID]int),
	}
}

func (T *Pool) NewSink(constraints rob.Constraints) *sink.Sink {
	snk := sink.NewSink()

	T.mu.Lock()
	defer T.mu.Unlock()

	T.sinks = append(T.sinks, sinkAndConstraints{
		sink:        snk,
		constraints: constraints,
	})

	i := 0
	for _, j := range T.backorder {
		if constraints.Satisfies(j.constraints) {
			snk.Queue(j.source, j.work)
		} else {
			T.backorder[i] = j
			i++
		}
	}
	T.backorder = T.backorder[:i]

	return snk
}

func (T *Pool) Schedule(source uuid.UUID, work any, constraints rob.Constraints) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if len(T.sinks) == 0 {
		T.backorder = append(T.backorder, job{
			source:      source,
			work:        work,
			constraints: constraints,
		})
		return
	}

	affinity, ok := T.affinity[source]
	if !ok {
		affinity = rand.Intn(len(T.sinks))
		T.affinity[source] = affinity
	}

	snk := T.sinks[affinity]
	if !snk.constraints.Satisfies(constraints) {
		// choose a new affinity that satisfies constraints
		ok = false
		for id, s := range T.sinks {
			if s.constraints.Satisfies(constraints) {
				snk = s
				affinity = id
				T.affinity[source] = affinity
				ok = true
				break
			}
		}
		if !ok {
			T.backorder = append(T.backorder, job{
				source:      source,
				work:        work,
				constraints: constraints,
			})
			return
		}
	}

	// yay, queued
	snk.sink.Queue(source, work)
}
