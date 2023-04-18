package race

import (
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
	}()
	if <-chans[7] != 1 {
		panic("expected to receive 1")
	}
}
