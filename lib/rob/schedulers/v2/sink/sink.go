package sink

import (
	"github.com/google/uuid"
	"log"
	"sync"
	"time"

	"pggat/lib/rob/schedulers/v2/job"
	"pggat/lib/util/rbtree"
	"pggat/lib/util/ring"
)

type Sink struct {
	id uuid.UUID

	// non final
	active uuid.UUID
	start  time.Time

	floor     time.Duration
	stride    map[uuid.UUID]time.Duration
	pending   map[uuid.UUID]*ring.Ring[job.Stalled]
	scheduled rbtree.RBTree[time.Duration, job.Stalled]

	mu sync.Mutex
}

func NewSink(id uuid.UUID) *Sink {
	return &Sink{
		id: id,
	}
}

func (T *Sink) schedule(j job.Stalled) bool {
	if T.active == j.User {
		log.Printf("couldn't schedule because user %v is active", j.User)
		panic("hmmmm")
		return false
	}

	stride, ok := T.stride[j.User]
	if !ok {
		// set to max
		stride = T.floor
		if s, _, ok := T.scheduled.Max(); ok {
			stride = s + 1
		}
		if T.stride == nil {
			T.stride = make(map[uuid.UUID]time.Duration)
		}
		T.stride[j.User] = stride
	} else if stride < T.floor {
		stride = T.floor
		T.stride[j.User] = stride
	}

	for {
		// find unique stride to schedule on
		s, ok := T.scheduled.Get(stride)
		if !ok {
			break
		}

		if s.User == j.User {
			log.Println("couldn't schedule because user is scheduled")
			return false
		}
		stride += 1
	}

	T.scheduled.Set(stride, j)
	return true
}

func (T *Sink) enqueue(j job.Stalled) {
	if T.active == uuid.Nil {
		// run it now
		T.acquire(j.User)
		j.Ready <- T.id
		return
	}

	if T.schedule(j) {
		return
	}

	p, ok := T.pending[j.User]

	// add to pending queue
	if !ok {
		p = ring.NewRing[job.Stalled](0, 1)
		if T.pending == nil {
			T.pending = make(map[uuid.UUID]*ring.Ring[job.Stalled])
		}
		T.pending[j.User] = p
	}

	p.PushBack(j)
}

func (T *Sink) Enqueue(j ...job.Stalled) {
	// enqueue job
	T.mu.Lock()
	defer T.mu.Unlock()

	for _, jj := range j {
		T.enqueue(jj)
	}
}

func (T *Sink) acquire(user uuid.UUID) {
	if T.active != uuid.Nil {
		panic("acquire called when already in use")
	}
	T.active = user
	T.start = time.Now()
}

func (T *Sink) Acquire(j job.Concurrent) bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.active != uuid.Nil {
		// already active
		return false
	}

	T.acquire(j.User)

	return true
}

func (T *Sink) enqueueNext(user uuid.UUID) {
	pending, ok := T.pending[user]
	if !ok {
		return
	}
	j, ok := pending.PopFront()
	if !ok {
		return
	}
	if ok = T.schedule(j); !ok {
		pending.PushFront(j)
		return
	}
}

func (T *Sink) next() bool {
	now := time.Now()
	if T.active != uuid.Nil {
		user := T.active
		dur := now.Sub(T.start)
		T.active = uuid.Nil
		T.start = now

		if T.stride == nil {
			T.stride = make(map[uuid.UUID]time.Duration)
		}
		T.stride[user] += dur

		T.enqueueNext(user)
	}

	stride, j, ok := T.scheduled.Min()
	if !ok {
		return false
	}
	T.scheduled.Delete(stride)
	if stride > T.floor {
		T.floor = stride
	}

	T.acquire(j.User)
	j.Ready <- T.id
	return true
}

func (T *Sink) Release() (hasMore bool) {
	T.mu.Lock()
	defer T.mu.Unlock()

	return T.next()
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

func (T *Sink) StealFor(rhs *Sink) uuid.UUID {
	if T == rhs {
		return uuid.Nil
	}

	T.mu.Lock()

	stride, j, ok := T.scheduled.Min()
	if !ok {
		T.mu.Unlock()
		return uuid.Nil
	}
	T.scheduled.Delete(stride)

	user := j.User

	pending := T.pending[user]
	delete(T.pending, user)

	T.mu.Unlock()

	rhs.mu.Lock()
	defer rhs.mu.Unlock()
	rhs.enqueue(j)

	if pending != nil {
		for j, ok = pending.PopFront(); ok; j, ok = pending.PopFront() {
			rhs.enqueue(j)
		}
	}

	return user
}

func (T *Sink) RemoveUser(user uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	delete(T.pending, user)
	delete(T.stride, user)
}

func (T *Sink) IsScheduled(user uuid.UUID) bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	for s, j, ok := T.scheduled.Min(); ok; s, j, ok = T.scheduled.Next(s) {
		if j.User == user {
			return true
		}
	}

	return false
}

func (T *Sink) IsPending(user uuid.UUID) bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	p, ok := T.pending[user]
	if !ok {
		return false
	}

	return p.Length() > 0
}
