package schedulers

import (
	"runtime"
	"sync"
	"testing"
	"time"

	"pggat2/lib/rob"
)

type Work struct {
	Sender      int
	Duration    time.Duration
	Done        chan<- struct{}
	Constraints rob.Constraints
}

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

	v, _ := T.table[user]
	return v
}

func testSink(sched *Scheduler, table *ShareTable, constraints rob.Constraints) {
	sink := sched.NewSink(constraints)
	for {
		w := sink.Read()
		switch v := w.(type) {
		case Work:
			if !constraints.Satisfies(v.Constraints) {
				panic("Scheduler did not obey constraints")
			}
			// dummy load
			start := time.Now()
			for time.Since(start) < v.Duration {
			}
			table.Inc(v.Sender)
			close(v.Done)
		}
	}
}

func testSource(sched *Scheduler, id int, dur time.Duration, constraints rob.Constraints) {
	source := sched.NewSource()
	for {
		done := make(chan struct{})
		w := Work{
			Sender:      id,
			Duration:    dur,
			Done:        done,
			Constraints: constraints,
		}
		source.Schedule(w, constraints)
		<-done
	}
}

func similar(v0, v1 int, vn ...int) bool {
	const margin = 0.05 // 5% margin of error

	min := v0
	max := v0

	if v1 < min {
		min = v1
	}
	if v1 > max {
		max = v1
	}

	for _, v := range vn {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	if (float64(max-min) / float64(max)) > margin {
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
	sched := NewScheduler()
	go testSink(sched, &table, 0)

	go testSource(sched, 0, 10*time.Millisecond, 0)
	go testSource(sched, 1, 10*time.Millisecond, 0)
	go testSource(sched, 2, 50*time.Millisecond, 0)
	go testSource(sched, 3, 100*time.Millisecond, 0)

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
	sched := NewScheduler()
	go testSink(sched, &table, 0)

	go testSource(sched, 0, 10*time.Millisecond, 0)
	go testSource(sched, 1, 10*time.Millisecond, 0)

	time.Sleep(10 * time.Second)

	go testSource(sched, 2, 10*time.Millisecond, 0)
	go testSource(sched, 3, 10*time.Millisecond, 0)

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
	sched := NewScheduler()
	go testSink(sched, &table, 0)
	go testSink(sched, &table, 0)

	go testSource(sched, 0, 10*time.Millisecond, 0)
	go testSource(sched, 1, 10*time.Millisecond, 0)
	go testSource(sched, 2, 10*time.Millisecond, 0)
	go testSource(sched, 3, 10*time.Millisecond, 0)

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
	}

	if t0 == 0 {
		t.Error("expected executions on all sources (is there a race in the balancer??)")
		t.Errorf("%s", allStacks())
	}
}

func TestScheduler_StealUnbalanced(t *testing.T) {
	var table ShareTable
	sched := NewScheduler()
	go testSink(sched, &table, 0)
	go testSink(sched, &table, 0)

	go testSource(sched, 0, 10*time.Millisecond, 0)
	go testSource(sched, 1, 10*time.Millisecond, 0)
	go testSource(sched, 2, 10*time.Millisecond, 0)

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
	}

	if t0 == 0 {
		t.Error("expected executions on all sources (is there a race in the balancer??)")
		t.Errorf("%s", allStacks())
	}
}

func TestScheduler_Constraints(t *testing.T) {
	const (
		ConstraintA rob.Constraints = 1 << iota
		ConstraintB
	)

	var table ShareTable
	sched := NewScheduler()

	go testSink(sched, &table, rob.Constraints.All(ConstraintA, ConstraintB))
	go testSink(sched, &table, ConstraintA)
	go testSink(sched, &table, ConstraintB)

	go testSource(sched, 0, 10*time.Millisecond, rob.Constraints.All(ConstraintA, ConstraintB))
	go testSource(sched, 1, 10*time.Millisecond, rob.Constraints.All(ConstraintA, ConstraintB))
	go testSource(sched, 2, 10*time.Millisecond, ConstraintA)
	go testSource(sched, 3, 10*time.Millisecond, ConstraintA)
	go testSource(sched, 4, 10*time.Millisecond, ConstraintB)
	go testSource(sched, 5, 10*time.Millisecond, ConstraintB)

	time.Sleep(20 * time.Second)
	t0 := table.Get(0)
	t1 := table.Get(1)
	t2 := table.Get(2)
	t3 := table.Get(3)
	t4 := table.Get(4)
	t5 := table.Get(5)

	/*
		Expectations:
		- all users should get similar # of executions (shares of 0 and 1 may be less because they have less sinks they can use: 1 vs 2)
		- all constraints should be honored
	*/

	t.Log("share of 0:", t0)
	t.Log("share of 1:", t1)
	t.Log("share of 2:", t2)
	t.Log("share of 3:", t3)
	t.Log("share of 4:", t4)
	t.Log("share of 5:", t5)
}
