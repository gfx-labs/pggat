package slices

import (
	"testing"
)

func assertLength[T any](t *testing.T, slice []T, length int) {
	if len(slice) != length {
		t.Error("expected length to be", length, "but got", len(slice))
	}
}

func TestResize_Grow(t *testing.T) {
	x := make([]byte, 0, 2)
	x = Resize(x, 5)
	assertLength(t, x, 5)
	x = Resize(x, 10)
	assertLength(t, x, 10)
}

func TestResize_Reslice(t *testing.T) {
	initial := make([]byte, 1, 16)
	x := initial
	x = Resize(x, 10)
	assertLength(t, x, 10)
	x = Resize(x, 4)
	assertLength(t, x, 4)
	x = Resize(x, 16)
	assertLength(t, x, 16)
	x = Resize(x, 1)
	assertLength(t, x, 1)
	if &initial[0] != &x[0] {
		t.Error("slice was re-allocated")
	}
}
