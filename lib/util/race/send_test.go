package race

import (
	"math/rand"
	"testing"
)

func TestSend(t *testing.T) {
	var chans []chan int
	for i := 0; i < 10; i++ {
		chans = append(chans, make(chan int))
	}
	go func() {
		Send(func(i int) (chan<- int, bool) {
			if i >= len(chans) {
				return nil, false
			}
			return chans[i], true
		}, 1)
		Send(func(i int) (chan<- int, bool) {
			if i >= len(chans) {
				return nil, false
			}
			return chans[i], true
		}, 2)
		Send(func(i int) (chan<- int, bool) {
			if i >= len(chans) {
				return nil, false
			}
			return chans[i], true
		}, 3)
	}()
	if <-chans[7] != 1 {
		panic("expected to receive 1")
	}
	if <-chans[3] != 2 {
		panic("expected to receive 2")
	}
	if <-chans[5] != 3 {
		panic("expected to receive 3")
	}
}

func BenchmarkSend(b *testing.B) {
	var chans []chan int
	for i := 0; i < 10; i++ {
		chans = append(chans, make(chan int))
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		v := rand.Int()
		go func() {
			if <-chans[rand.Intn(10)] != v {
				panic("expected correct value")
			}
		}()
		Send(func(i int) (chan<- int, bool) {
			if i >= len(chans) {
				return nil, false
			}
			return chans[i], true
		}, v)
	}
}

func BenchmarkSend_Raw(b *testing.B) {
	var chans []chan int
	for i := 0; i < 10; i++ {
		chans = append(chans, make(chan int))
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		v := rand.Int()
		go func() {
			if <-chans[rand.Intn(10)] != v {
				panic("expected correct value")
			}
		}()
		// Send above is equal to the following
		select {
		case chans[0] <- v:
		case chans[1] <- v:
		case chans[2] <- v:
		case chans[3] <- v:
		case chans[4] <- v:
		case chans[5] <- v:
		case chans[6] <- v:
		case chans[7] <- v:
		case chans[8] <- v:
		case chans[9] <- v:
		}
	}
}
