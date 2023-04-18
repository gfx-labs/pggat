package race

import (
	"testing"
)

func TestRecv(t *testing.T) {
	var chans []chan int
	for i := 0; i < 10; i++ {
		chans = append(chans, make(chan int))
	}
	go func() {
		v, ok := Recv(func(i int) (<-chan int, bool) {
			if i >= len(chans) {
				return nil, false
			}
			return chans[i], true
		})
		if !ok || v != 1234 {
			panic("expected to receive 1234")
		}
	}()
	chans[1] <- 1234
}
