package v0

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

func TestScheduler(t *testing.T) {
	var table ShareTable
	sched := NewScheduler()
	go func() {
		sink := sched.NewSink()
		for {
			w := sink.Read()
			switch v := w.(type) {
			case Work:
				<-time.After(v.Duration)
				table.Inc(v.Sender)
				close(v.Done)
			}
		}
	}()
	go func() {
		source := sched.NewSource()
		for {
			done := make(chan struct{})
			w := Work{
				Sender:   0,
				Duration: 10 * time.Millisecond,
				Done:     done,
			}
			source.Schedule(w)
			<-done
		}
	}()
	go func() {
		source := sched.NewSource()
		for {
			done := make(chan struct{})
			w := Work{
				Sender:   3,
				Duration: 10 * time.Millisecond,
				Done:     done,
			}
			source.Schedule(w)
			<-done
		}
	}()
	go func() {
		source := sched.NewSource()
		for {
			done := make(chan struct{})
			w := Work{
				Sender:   1,
				Duration: 100 * time.Millisecond,
				Done:     done,
			}
			source.Schedule(w)
			<-done
		}
	}()
	go func() {
		source := sched.NewSource()
		for {
			done := make(chan struct{})
			w := Work{
				Sender:   2,
				Duration: 50 * time.Millisecond,
				Done:     done,
			}
			source.Schedule(w)
			<-done
		}
	}()

	<-time.After(20 * time.Second)
	t0 := table.Get(0)
	t1 := table.Get(1)
	t2 := table.Get(2)
	t3 := table.Get(3)
	log.Println("share of 0:", t0)
	log.Println("share of 1:", t1)
	log.Println("share of 2:", t2)
	log.Println("share of 3:", t3)
}
