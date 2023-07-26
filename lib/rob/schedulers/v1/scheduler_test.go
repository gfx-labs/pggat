package schedulers

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/rob"
)

type Work struct {
	Sender   int
	Duration time.Duration
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

type TestSink struct {
	table       *ShareTable
	constraints rob.Constraints
	inuse       atomic.Bool
	remove      atomic.Bool
	removed     atomic.Bool
}

func (T *TestSink) Do(ctx *rob.Context, work any) {
	if T.inuse.Swap(true) {
		panic("Sink was already inuse")
	}
	defer T.inuse.Store(false)
	if !T.constraints.Satisfies(ctx.Constraints) {
		panic("Scheduler did not obey constraints")
	}
	v := work.(Work)
	start := time.Now()
	for time.Since(start) < v.Duration {
	}
	T.table.Inc(v.Sender)
	if T.remove.Load() {
		removed := T.removed.Swap(true)
		if removed {
			panic("Scheduler did not remove when requested")
		}
		ctx.Remove()
	}
}

var _ rob.Worker = (*TestSink)(nil)

func testSink(sched *Scheduler, table *ShareTable, constraints rob.Constraints) uuid.UUID {
	return sched.AddSink(constraints, &TestSink{
		table:       table,
		constraints: constraints,
	})
}

func testSinkRemoveAfter(sched *Scheduler, table *ShareTable, constraints rob.Constraints, removeAfter time.Duration) uuid.UUID {
	sink := &TestSink{
		table:       table,
		constraints: constraints,
	}
	go func() {
		time.Sleep(removeAfter)
		sink.remove.Store(true)
	}()
	return sched.AddSink(constraints, sink)
}

func testSource(sched *Scheduler, id int, dur time.Duration, constraints rob.Constraints) {
	source := sched.NewSource()
	for {
		w := Work{
			Sender:   id,
			Duration: dur,
		}
		source.Do(&rob.Context{
			Constraints: constraints,
		}, w)
	}
}

func testStarver(sched *Scheduler, id int, dur time.Duration, constraints rob.Constraints) {
	for {
		source := sched.NewSource()
		w := Work{
			Sender:   id,
			Duration: dur,
		}
		source.Do(&rob.Context{
			Constraints: constraints,
		}, w)
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
	testSink(sched, &table, 0)

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
	testSink(sched, &table, 0)

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
	testSink(sched, &table, 0)
	testSink(sched, &table, 0)

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
	testSink(sched, &table, 0)
	testSink(sched, &table, 0)

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

	if t0 == 0 || t1 == 0 || t2 == 0 {
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

	testSink(sched, &table, rob.Constraints.All(ConstraintA, ConstraintB))
	testSink(sched, &table, ConstraintA)
	testSink(sched, &table, ConstraintB)

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

func TestScheduler_IdleWake(t *testing.T) {
	var table ShareTable
	sched := NewScheduler()

	testSink(sched, &table, 0)

	time.Sleep(10 * time.Second)

	go testSource(sched, 0, 10*time.Millisecond, 0)

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
	sched := NewScheduler()

	go testSource(sched, 0, 10*time.Millisecond, 0)

	time.Sleep(10 * time.Second)

	testSink(sched, &table, 0)

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
	sched := NewScheduler()

	testSink(sched, &table, 0)

	go testStarver(sched, 1, 10*time.Millisecond, 0)
	go testStarver(sched, 2, 10*time.Millisecond, 0)
	go testSource(sched, 0, 10*time.Millisecond, 0)

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
	sched := NewScheduler()
	testSink(sched, &table, 0)
	toRemove := testSink(sched, &table, 0)

	go testSource(sched, 0, 10*time.Millisecond, 0)
	go testSource(sched, 1, 10*time.Millisecond, 0)
	go testSource(sched, 2, 10*time.Millisecond, 0)
	go testSource(sched, 3, 10*time.Millisecond, 0)

	time.Sleep(10 * time.Second)

	sched.RemoveSink(toRemove)

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

func TestScheduler_RemoveSinkInner(t *testing.T) {
	var table ShareTable
	sched := NewScheduler()
	testSink(sched, &table, 0)
	testSinkRemoveAfter(sched, &table, 0, 10*time.Second)

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
