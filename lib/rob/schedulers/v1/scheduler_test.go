package schedulers

import (
	"log"
	"sync"
	"testing"
	"time"
)

type Work struct {
	Sender   int
	Duration time.Duration
	Done     chan<- struct{}
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

func testSink(sched *Scheduler, table *ShareTable) {
	sink := sched.NewSink()
	for {
		w := sink.Read()
		switch v := w.(type) {
		case Work:
			time.Sleep(v.Duration)
			table.Inc(v.Sender)
			close(v.Done)
		}
	}
}

func testSource(sched *Scheduler, id int, dur time.Duration) {
	source := sched.NewSource()
	for {
		done := make(chan struct{})
		w := Work{
			Sender:   id,
			Duration: dur,
			Done:     done,
		}
		source.Schedule(w)
		<-done
	}
}

func TestScheduler(t *testing.T) {
	var table ShareTable
	sched := NewScheduler()
	go testSink(sched, &table)

	go testSource(sched, 0, 10*time.Millisecond)
	go testSource(sched, 1, 10*time.Millisecond)
	go testSource(sched, 2, 50*time.Millisecond)
	go testSource(sched, 3, 100*time.Millisecond)

	time.Sleep(20 * time.Second)
	t0 := table.Get(0)
	t1 := table.Get(1)
	t2 := table.Get(2)
	t3 := table.Get(3)
	log.Println("share of 0:", t0)
	log.Println("share of 1:", t1)
	log.Println("share of 2:", t2)
	log.Println("share of 3:", t3)
	log.Println("total:",
		time.Duration((t0+t1)*10)*time.Millisecond+
			time.Duration(t2*50)*time.Millisecond+
			time.Duration(t3*100)*time.Millisecond,
	)
}
