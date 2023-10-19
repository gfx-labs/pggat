package schedulers

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/rob"
	"gfx.cafe/gfx/pggat/lib/util/pools"
	"gfx.cafe/gfx/pggat/lib/util/rbtree"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Scheduler struct {
	cc pools.Locked[chan uuid.UUID]

	closed bool

	queue   []*Worker
	workers map[uuid.UUID]*Worker

	floor    time.Duration
	users    map[uuid.UUID]*User
	schedule rbtree.RBTree[time.Duration, Job]

	mu sync.Mutex
}

func (T *Scheduler) AddWorker(id uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return
	}

	if T.workers == nil {
		T.workers = make(map[uuid.UUID]*Worker)
	}
	worker := &Worker{
		ID: id,
	}
	T.workers[id] = worker

	T.release(worker)
}

func (T *Scheduler) DeleteWorker(worker uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return
	}

	w, ok := T.workers[worker]
	if !ok {
		return
	}
	delete(T.workers, worker)
	T.queue = slices.Delete(T.queue, w)
}

func (T *Scheduler) AddUser(user uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return
	}

	if T.users == nil {
		T.users = make(map[uuid.UUID]*User)
	}

	stride := T.floor
	if s, _, ok := T.schedule.Max(); ok {
		stride = s + 1
	}

	T.users[user] = &User{
		ID:     user,
		Stride: stride,
	}
}

func (T *Scheduler) DeleteUser(user uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return
	}

	u, ok := T.users[user]
	if !ok {
		return
	}
	delete(T.users, user)

	if u.Scheduled {
		var j Job
		j, ok = T.schedule.Get(u.Stride)
		if ok {
			close(j.Ready)
			T.schedule.Delete(u.Stride)
		}
	}
}

func (T *Scheduler) Acquire(user uuid.UUID, sync rob.SyncMode) uuid.UUID {
	v, c := func() (uuid.UUID, chan uuid.UUID) {
		T.mu.Lock()
		defer T.mu.Unlock()

		if T.closed {
			return uuid.Nil, nil
		}

		u, ok := T.users[user]
		if !ok {
			return uuid.Nil, nil
		}

		if len(T.queue) > 0 {
			worker := T.queue[len(T.queue)-1]
			T.queue = T.queue[:len(T.queue)-1]

			u.Worker = worker
			worker.User = u
			worker.Since = time.Now()

			return worker.ID, nil
		}

		if sync == rob.SyncModeNonBlocking {
			return uuid.Nil, nil
		}

		ready, _ := T.cc.Get()
		if ready == nil {
			ready = make(chan uuid.UUID, 1)
		}
		job := Job{
			User:  u,
			Ready: ready,
		}

		// find empty slot
		if u.Stride < T.floor {
			u.Stride = T.floor
		}
		for _, ok = T.schedule.Get(u.Stride); ok; _, ok = T.schedule.Get(u.Stride) {
			u.Stride++
		}
		T.schedule.Set(u.Stride, job)
		u.Scheduled = true

		return uuid.Nil, ready
	}()

	if v != uuid.Nil {
		return v
	}

	if c != nil {
		var ok bool
		v, ok = <-c
		if ok {
			T.cc.Put(c)
		}
	}

	return v
}

func (T *Scheduler) release(worker *Worker) {
	now := time.Now()

	// update prev user and state
	if worker.User != nil {
		worker.User.Stride += now.Sub(worker.Since)
		worker.User.Worker = nil

		worker.Since = now
		worker.User = nil
	}

	// try to give it to the next pending
	stride, job, ok := T.schedule.Min()
	if !ok {
		// no work available, append to queue
		T.queue = append(T.queue, worker)
		return
	}

	T.floor = stride
	T.schedule.Delete(stride)

	job.User.Worker = worker
	job.User.Scheduled = false

	worker.Since = now
	worker.User = job.User

	job.Ready <- worker.ID

	return
}

func (T *Scheduler) Release(worker uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return
	}

	w, ok := T.workers[worker]
	if !ok {
		return
	}

	T.release(w)
}

func (T *Scheduler) Close() {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.closed {
		return
	}

	T.closed = true

	T.users = nil
	T.workers = nil
	for stride, job, ok := T.schedule.Min(); ok; stride, job, ok = T.schedule.Min() {
		T.schedule.Delete(stride)
		close(job.Ready)
	}
	T.queue = nil
}

var _ rob.Scheduler = (*Scheduler)(nil)
