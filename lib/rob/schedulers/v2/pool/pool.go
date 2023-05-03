package pool

import (
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/rob/schedulers/v2/job"
	"pggat2/lib/rob/schedulers/v2/queue"
)

type constrainedQueue struct {
	queue       *queue.Queue
	constraints rob.Constraints
}

type Pool struct {
	affinity  map[uuid.UUID]uuid.UUID
	queues    map[uuid.UUID]constrainedQueue
	backorder []job.Job
	mu        sync.Mutex
}

func MakePool() Pool {
	return Pool{
		affinity: make(map[uuid.UUID]uuid.UUID),
		queues:   make(map[uuid.UUID]constrainedQueue),
	}
}

func (T *Pool) NewQueue(id uuid.UUID, constraints rob.Constraints) *queue.Queue {
	q := queue.NewQueue()

	T.mu.Lock()
	defer T.mu.Unlock()

	T.queues[id] = constrainedQueue{
		queue:       q,
		constraints: constraints,
	}

	i := 0
	for _, j := range T.backorder {
		if constraints.Satisfies(j.Constraints) {
			q.Queue(j)
		} else {
			T.backorder[i] = j
			i++
		}
	}
	T.backorder = T.backorder[:i]

	return q
}

func (T *Pool) Schedule(work job.Job) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if len(T.queues) == 0 {
		T.backorder = append(T.backorder, work)
		return
	}

	var q constrainedQueue
	affinity, ok := T.affinity[work.Source]
	if ok {
		q = T.queues[affinity]
	}

	if !ok || !q.constraints.Satisfies(work.Constraints) || !q.queue.Idle() {
		// choose a new affinity that satisfies constraints
		ok = false
		for id, s := range T.queues {
			if s.constraints.Satisfies(work.Constraints) {
				current := id == affinity
				q = s
				affinity = id
				ok = true
				if !current && s.queue.Idle() {
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
	q.queue.Queue(work)
}

func (T *Pool) StealFor(id uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	q, ok := T.queues[id]
	if !ok {
		return
	}

	for _, s := range T.queues {
		if s == q {
			continue
		}
		works, ok := s.queue.Steal(q.constraints)
		if !ok {
			continue
		}
		if len(works) > 0 {
			source := works[0].Source
			T.affinity[source] = id
		}
		for _, work := range works {
			q.queue.Queue(work)
		}
		break
	}
}
