package race

import (
	"testing"

	"gfx.cafe/gfx/pggat/lib/util/iter"
)

func TestRecv(t *testing.T) {
	var chans []chan int
	for i := 0; i < 10; i++ {
		chans = append(chans, make(chan int))
	}
	go func() {
		v, ok := Recv(iter.Map(iter.Slice(chans), func(c chan int) <-chan int {
			return c
		}))
		if !ok || v != 1234 {
			panic("expected to receive 1234")
		}
	}()
	chans[1] <- 1234
}
