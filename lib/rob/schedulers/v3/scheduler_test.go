package schedulers

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/rob"
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

func testSource(sched *Scheduler, tab *ShareTable, id int, dur time.Duration) {
	source := uuid.New()
	sched.AddUser(source)
	for {
		sink := sched.Acquire(source, rob.SyncModeTryNonBlocking)
		start := time.Now()
		for time.Since(start) < dur {
			runtime.Gosched()
		}
		tab.Inc(id)
		sched.Release(sink)
	}
}

func testStarver(sched *Scheduler, tab *ShareTable, id int, dur time.Duration) {
	for {
		func() {
			source := uuid.New()
			sched.AddUser(source)
			defer sched.DeleteUser(source)

			sink := sched.Acquire(source, rob.SyncModeTryNonBlocking)
			defer sched.Release(sink)
			start := time.Now()
			for time.Since(start) < dur {
				runtime.Gosched()
			}
			tab.Inc(id)
		}()
	}
}

func similar(v0, v1 int, vn ...int) bool {
	const margin = 0.25 // 25% margin of error

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

	if (float64(maximum-minimum) / float64(maximum)) > margin {
		return false
	}
	return true
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
	var table ShareTable
	sched := new(Scheduler)
	testSink(sched)

	go testSource(sched, &table, 0, 10*time.Millisecond)
	go testSource(sched, &table, 1, 10*time.Millisecond)
	go testSource(sched, &table, 2, 50*time.Millisecond)
	go testSource(sched, &table, 3, 100*time.Millisecond)

	time.Sleep(20 * time.Second)
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
	var table ShareTable
	sched := new(Scheduler)
	testSink(sched)

	go testSource(sched, &table, 0, 10*time.Millisecond)
	go testSource(sched, &table, 1, 10*time.Millisecond)

	time.Sleep(10 * time.Second)

	go testSource(sched, &table, 2, 10*time.Millisecond)
	go testSource(sched, &table, 3, 10*time.Millisecond)

	time.Sleep(10 * time.Second)
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
	var table ShareTable
	sched := new(Scheduler)
	testSink(sched)
	testSink(sched)

	go testSource(sched, &table, 0, 10*time.Millisecond)
	go testSource(sched, &table, 1, 10*time.Millisecond)
	go testSource(sched, &table, 2, 10*time.Millisecond)
	go testSource(sched, &table, 3, 10*time.Millisecond)

	time.Sleep(20 * time.Second)
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
	var table ShareTable
	sched := new(Scheduler)
	testSink(sched)
	testSink(sched)

	go testSource(sched, &table, 0, 10*time.Millisecond)
	go testSource(sched, &table, 1, 10*time.Millisecond)
	go testSource(sched, &table, 2, 10*time.Millisecond)

	time.Sleep(20 * time.Second)
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
	var table ShareTable
	sched := new(Scheduler)

	testSink(sched)

	time.Sleep(10 * time.Second)

	go testSource(sched, &table, 0, 10*time.Millisecond)

	time.Sleep(10 * time.Second)
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
	var table ShareTable
	sched := new(Scheduler)

	go testSource(sched, &table, 0, 10*time.Millisecond)

	time.Sleep(10 * time.Second)

	testSink(sched)

	time.Sleep(10 * time.Second)
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
	var table ShareTable
	sched := new(Scheduler)

	testSink(sched)

	go testStarver(sched, &table, 1, 10*time.Millisecond)
	go testStarver(sched, &table, 2, 10*time.Millisecond)
	go testSource(sched, &table, 0, 10*time.Millisecond)

	time.Sleep(20 * time.Second)
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
	var table ShareTable
	sched := new(Scheduler)
	testSink(sched)
	toRemove := testSink(sched)

	go testSource(sched, &table, 0, 10*time.Millisecond)
	go testSource(sched, &table, 1, 10*time.Millisecond)
	go testSource(sched, &table, 2, 10*time.Millisecond)
	go testSource(sched, &table, 3, 10*time.Millisecond)

	time.Sleep(10 * time.Second)

	sched.DeleteWorker(toRemove)

	time.Sleep(10 * time.Second)

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
