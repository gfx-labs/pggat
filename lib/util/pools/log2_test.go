package pools

import (
	"testing"
)

func TestLog2_Get(t *testing.T) {
	pool := new(Log2[byte])
	x := pool.Get(123)
	if len(x) != 123 {
		t.Error("expected length to equal")
	}
	pool.Put(x)
	x2 := pool.Get(128)
	if &x[0] != &x2[0] {
		t.Error("expected to get the same slice back")
	}
}
