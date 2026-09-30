package schedulers

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type ShareTable struct {
	table map[int]int
	mu    sync.RWMutex
}

func (T *ShareTable) Inc(user int) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.table == nil {
		T.table = make(map[int]int)
	}
	T.table[user]++
}

func (T *ShareTable) Get(user int) int {
	T.mu.RLock()
	defer T.mu.RUnlock()

	return T.table[user]
}

func testSink(sched *Scheduler) uuid.UUID {
	id := uuid.New()
	sched.AddWorker(id)
	return id
}

// unit is the base job length. Windows are sized in units so sample counts stay constant.
const unit = time.Millisecond

// window is how long a test phase runs before shares are measured.
const window = 1000 * unit

func stopOnCleanup(t *testing.T) <-chan struct{} {
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })
	return done
}

func stopped(done <-chan struct{}) bool {
	select {
	case <-done:
		return true
	default:
		return false
	}
}

func testSource(done <-chan struct{}, sched *Scheduler, tab *ShareTable, id int, dur time.Duration) {
	source := uuid.New()
	sched.AddUser(source)
	for !stopped(done) {
		sink := sched.Acquire(source, 0)
		time.Sleep(dur)
		tab.Inc(id)
		sched.Release(sink)
	}
}

func testStarver(done <-chan struct{}, sched *Scheduler, tab *ShareTable, id int, dur time.Duration) {
	for !stopped(done) {
		func() {
			source := uuid.New()
			sched.AddUser(source)
			defer sched.DeleteUser(source)

			sink := sched.Acquire(source, 0)
			defer sched.Release(sink)
			time.Sleep(dur)
			tab.Inc(id)
		}()
	}
}

func similar(v0, v1 int, vn ...int) bool {
	const margin = 0.5 // 50% margin of error

	minimum := v0
	maximum := v0

	if v1 < minimum {
		minimum = v1
	}
	if v1 > maximum {
		maximum = v1
	}

	for _, v := range vn {
		if v < minimum {
			minimum = v
		}
		if v > maximum {
			maximum = v
		}
	}

	return (float64(maximum-minimum) / float64(maximum)) <= margin
}

// like debug.Stack but gets all stacks
func allStacks() []byte {
	buf := make([]byte, 1024)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return buf[:n]
		}
		buf = make([]byte, 2*len(buf))
	}
}

func TestScheduler(t *testing.T) {
	t.Parallel()
	done := stopOnCleanup(t)
	var table ShareTable
	sched := new(Scheduler)
	testSink(sched)

	go testSource(done, sched, &table, 0, unit)
	go testSource(done, sched, &table, 1, unit)
	go testSource(done, sched, &table, 2, 5*unit)
	go testSource(done, sched, &table, 3, 10*unit)

	time.Sleep(2 * window)
	t0 := table.Get(0)
	t1 := table.Get(1)
	t2 := table.Get(2)
	t3 := table.Get(3)

	/*
		Expectations:
		- 0 and 1 should be similar and have roughly 10x of 3
		- 2 should have about twice as many executions as 3
	*/

	t.Log("share of 0:", t0)
	t.Log("share of 1:", t1)
	t.Log("share of 2:", t2)
	t.Log("share of 3:", t3)

	if !similar(t0, t1) {
		t.Error("expected s0 and s1 to be similar")
	}

	if !similar(t0, t3*10) {
		t.Error("expected s0 and s3*10 to be similar")
	}

	if !similar(t2, t3*2) {
		t.Error("expected s2 and s3*2 to be similar")
	}
}

func TestScheduler_Late(t *testing.T) {
	t.Parallel()
	done := stopOnCleanup(t)
	var table ShareTable
	sched := new(Scheduler)
	testSink(sched)

	go testSource(done, sched, &table, 0, unit)
	go testSource(done, sched, &table, 1, unit)

	time.Sleep(window)

	go testSource(done, sched, &table, 2, unit)
	go testSource(done, sched, &table, 3, unit)

	time.Sleep(window)
	t0 := table.Get(0)
	t1 := table.Get(1)
	t2 := table.Get(2)
	t3 := table.Get(3)

	/*
		Expectations:
		- 0 and 1 should be similar
		- 2 and 3 should be similar
		- 0 and 1 should have roughly three times as many executions as 2 and 3
	*/

	t.Log("share of 0:", t0)
	t.Log("share of 1:", t1)
	t.Log("share of 2:", t2)
	t.Log("share of 3:", t3)

	if !similar(t0, t1) {
		t.Error("expected s0 and s1 to be similar")
	}

	if !similar(t2, t3) {
		t.Error("expected s2 and s3 to be similar")
	}

	if !similar(t0, 3*t2) {
		t.Error("expected s0 and s2*3 to be similar")
	}
}

func TestScheduler_StealBalanced(t *testing.T) {
	t.Parallel()
	done := stopOnCleanup(t)
	var table ShareTable
	sched := new(Scheduler)
	testSink(sched)
	testSink(sched)

	go testSource(done, sched, &table, 0, unit)
	go testSource(done, sched, &table, 1, unit)
	go testSource(done, sched, &table, 2, unit)
	go testSource(done, sched, &table, 3, unit)

	time.Sleep(2 * window)
	t0 := table.Get(0)
	t1 := table.Get(1)
	t2 := table.Get(2)
	t3 := table.Get(3)

	/*
		Expectations:
		- all users should get similar # of executions
	*/

	t.Log("share of 0:", t0)
	t.Log("share of 1:", t1)
	t.Log("share of 2:", t2)
	t.Log("share of 3:", t3)

	if !similar(t0, t1, t2, t3) {
		t.Error("expected all shares to be similar")
		t.Errorf("%s", allStacks())
	}

	if t0 == 0 {
		t.Error("expected executions on all sources (is there a race in the balancer??)")
		t.Errorf("%s", allStacks())
	}
}

func TestScheduler_StealUnbalanced(t *testing.T) {
	t.Parallel()
	done := stopOnCleanup(t)
	var table ShareTable
	sched := new(Scheduler)
	testSink(sched)
	testSink(sched)

	go testSource(done, sched, &table, 0, unit)
	go testSource(done, sched, &table, 1, unit)
	go testSource(done, sched, &table, 2, unit)

	time.Sleep(2 * window)
	t0 := table.Get(0)
	t1 := table.Get(1)
	t2 := table.Get(2)

	/*
		Expectations:
		- all users should get similar # of executions
	*/

	t.Log("share of 0:", t0)
	t.Log("share of 1:", t1)
	t.Log("share of 2:", t2)

	if !similar(t0, t1, t2) {
		t.Error("expected all shares to be similar")
		t.Errorf("%s", allStacks())
	}

	if t0 == 0 || t1 == 0 || t2 == 0 {
		t.Error("expected executions on all sources (is there a race in the balancer??)")
		t.Errorf("%s", allStacks())
	}
}

func TestScheduler_IdleWake(t *testing.T) {
	t.Parallel()
	done := stopOnCleanup(t)
	var table ShareTable
	sched := new(Scheduler)

	testSink(sched)

	time.Sleep(window)

	go testSource(done, sched, &table, 0, unit)

	time.Sleep(window)
	t0 := table.Get(0)

	/*
		Expectations:
		- 0 should have some executions
	*/

	if t0 == 0 {
		t.Error("expected executions to be greater than 0 (is idle waking broken?)")
	}

	t.Log("share of 0:", t0)
}

func TestScheduler_LateSink(t *testing.T) {
	t.Parallel()
	done := stopOnCleanup(t)
	var table ShareTable
	sched := new(Scheduler)

	go testSource(done, sched, &table, 0, unit)

	time.Sleep(window)

	testSink(sched)

	time.Sleep(window)
	t0 := table.Get(0)

	/*
		Expectations:
		- 0 should have some executions
	*/

	if t0 == 0 {
		t.Error("expected executions to be greater than 0 (is backlog broken?)")
	}

	t.Log("share of 0:", t0)
}

func TestScheduler_Starve(t *testing.T) {
	t.Parallel()
	done := stopOnCleanup(t)
	var table ShareTable
	sched := new(Scheduler)

	testSink(sched)

	go testStarver(done, sched, &table, 1, unit)
	go testStarver(done, sched, &table, 2, unit)
	go testSource(done, sched, &table, 0, unit)

	time.Sleep(2 * window)
	t0 := table.Get(0)
	t1 := table.Get(1)
	t2 := table.Get(2)

	/*
		Expectations:
		- 0 should not be starved
	*/

	t.Log("share of 0:", t0)
	t.Log("share of 1:", t1)
	t.Log("share of 2:", t2)

	if !similar(t0, t1, t2) {
		t.Error("expected all executions to be similar (is 0 starving?)")
	}
}

func TestScheduler_RemoveSinkOuter(t *testing.T) {
	t.Parallel()
	done := stopOnCleanup(t)
	var table ShareTable
	sched := new(Scheduler)
	testSink(sched)
	toRemove := testSink(sched)

	go testSource(done, sched, &table, 0, unit)
	go testSource(done, sched, &table, 1, unit)
	go testSource(done, sched, &table, 2, unit)
	go testSource(done, sched, &table, 3, unit)

	time.Sleep(window)

	sched.DeleteWorker(toRemove)

	time.Sleep(window)

	t0 := table.Get(0)
	t1 := table.Get(1)
	t2 := table.Get(2)
	t3 := table.Get(3)

	/*
		Expectations:
		- all users should get similar # of executions
	*/

	t.Log("share of 0:", t0)
	t.Log("share of 1:", t1)
	t.Log("share of 2:", t2)
	t.Log("share of 3:", t3)

	if !similar(t0, t1, t2, t3) {
		t.Error("expected all shares to be similar")
	}

	if t0 == 0 {
		t.Error("expected executions on all sources (is there a race in the balancer??)")
		t.Errorf("%s", allStacks())
	}
}
