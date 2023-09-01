package ring

import (
	ring2 "container/ring"
	"testing"
)

func assertSome[T comparable](t *testing.T, f func() (T, bool), value T) {
	v, ok := f()
	if !ok {
		t.Error("expected items but got nothing")
		return
	}
	if v != value {
		t.Error("expected", value, "but got", v)
		return
	}
}

func assertNone[T any](t *testing.T, f func() (T, bool)) {
	v, ok := f()
	if ok {
		t.Error("expected no items but found", v)
		return
	}
}

func assertGetSome[T comparable](t *testing.T, ring *Ring[T], index int, value T) {
	v, ok := ring.Get(index)
	if !ok {
		t.Error("expected items but got nothing")
		return
	}
	if v != value {
		t.Error("expected", value, "but got", v)
		return
	}
}

func assertGetNone[T comparable](t *testing.T, ring *Ring[T], index int) {
	v, ok := ring.Get(index)
	if ok {
		t.Error("expected nothing but got", v)
		return
	}
}

func assertLength[T any](t *testing.T, ring *Ring[T], length int) {
	l := ring.Length()
	if length != l {
		t.Error("expected length to be", length, "but got", l)
	}
}

func assertCapacity[T any](t *testing.T, ring *Ring[T], capacity int) {
	c := ring.Capacity()
	if capacity != c {
		t.Error("expected capacity to be", capacity, "but got", c)
	}
}

func TestRing_Get(t *testing.T) {
	r := new(Ring[int])
	r.PushBack(2)
	r.PushBack(3)
	r.PushBack(4)
	r.PushFront(1)
	assertGetSome(t, r, 0, 1)
	assertGetSome(t, r, 1, 2)
	assertGetSome(t, r, 2, 3)
	assertGetSome(t, r, 3, 4)
	r.PopBack()
	assertGetNone(t, r, 3)
	assertGetNone(t, r, -1)
}

func TestRing_New(t *testing.T) {
	r := new(Ring[int])
	r.PushBack(1)
	r.PushBack(2)
	r.PushBack(3)
	r.PushBack(4)

	assertLength(t, r, 4)

	assertSome(t, r.PopBack, 4)
	assertSome(t, r.PopBack, 3)
	assertSome(t, r.PopBack, 2)
	assertSome(t, r.PopBack, 1)
	assertNone(t, r.PopBack)

	assertLength(t, r, 0)
}

func TestRing_Back(t *testing.T) {
	r := MakeRing[int](0, 16)
	r.PushBack(1)
	r.PushBack(2)
	r.PushBack(3)
	r.PushBack(4)

	assertLength(t, &r, 4)

	assertSome(t, r.PopBack, 4)
	assertSome(t, r.PopBack, 3)
	assertSome(t, r.PopBack, 2)
	assertSome(t, r.PopBack, 1)
	assertNone(t, r.PopBack)

	assertLength(t, &r, 0)
}

func TestRing_Front(t *testing.T) {
	r := MakeRing[int](0, 16)
	r.PushFront(1)
	r.PushFront(2)
	r.PushFront(3)
	r.PushFront(4)

	assertLength(t, &r, 4)

	assertSome(t, r.PopFront, 4)
	assertSome(t, r.PopFront, 3)
	assertSome(t, r.PopFront, 2)
	assertSome(t, r.PopFront, 1)
	assertNone(t, r.PopBack)

	assertLength(t, &r, 0)
}

func TestRing_FrontBack(t *testing.T) {
	r := MakeRing[int](0, 16)
	r.PushFront(1)
	r.PushFront(2)
	r.PushFront(3)
	r.PushFront(4)

	assertLength(t, &r, 4)

	assertSome(t, r.PopBack, 1)
	assertSome(t, r.PopBack, 2)
	assertSome(t, r.PopBack, 3)
	assertSome(t, r.PopBack, 4)
	assertNone(t, r.PopBack)

	assertLength(t, &r, 0)
}

func TestRing_BackFront(t *testing.T) {
	r := MakeRing[int](0, 16)
	r.PushBack(1)
	r.PushBack(2)
	r.PushBack(3)
	r.PushBack(4)

	assertLength(t, &r, 4)

	assertSome(t, r.PopFront, 1)
	assertSome(t, r.PopFront, 2)
	assertSome(t, r.PopFront, 3)
	assertSome(t, r.PopFront, 4)
	assertNone(t, r.PopFront)

	assertLength(t, &r, 0)
}

func TestRing_Underflow(t *testing.T) {
	r := MakeRing[int](0, 16)
	assertNone(t, r.PopFront)
	assertNone(t, r.PopFront)
	assertNone(t, r.PopFront)
	assertNone(t, r.PopFront)

	assertLength(t, &r, 0)
}

func TestRing_Overflow(t *testing.T) {
	r := MakeRing[int](0, 16)
	assertNone(t, r.PopBack)
	assertNone(t, r.PopBack)
	assertNone(t, r.PopBack)
	assertNone(t, r.PopBack)

	assertLength(t, &r, 0)
}

// ensure no smashing or resizing when we put the exact amount in the ring
func TestRing_Glove(t *testing.T) {
	r := MakeRing[int](0, 4)
	r.PushBack(1)
	r.PushBack(2)
	r.PushBack(3)
	r.PushBack(4)

	assertLength(t, &r, 4)

	assertSome(t, r.PopFront, 1)
	assertSome(t, r.PopFront, 2)
	assertSome(t, r.PopFront, 3)
	assertSome(t, r.PopFront, 4)
	assertNone(t, r.PopFront)

	assertLength(t, &r, 0)
	assertCapacity(t, &r, 4)
}

// test case where tail pointer smashes into head pointer
func TestRing_SmashForward(t *testing.T) {
	r := MakeRing[int](0, 4)
	r.PushBack(1)
	r.PushBack(2)
	r.PushBack(3)
	r.PushBack(4)
	r.PushBack(5) // SMASH

	assertLength(t, &r, 5)

	// ensure we can still read out our values (ring should've resized)
	assertSome(t, r.PopFront, 1)
	assertSome(t, r.PopFront, 2)
	assertSome(t, r.PopFront, 3)
	assertSome(t, r.PopFront, 4)
	assertSome(t, r.PopFront, 5)
	assertNone(t, r.PopFront)

	assertLength(t, &r, 0)
}

// test case where head pointer smashes into tail pointer
func TestRing_SmashBackward(t *testing.T) {
	r := MakeRing[int](0, 4)
	r.PushFront(1)
	r.PushFront(2)
	r.PushFront(3)
	r.PushFront(4)
	r.PushFront(5) // SMASH

	assertLength(t, &r, 5)

	// ensure we can still read out our values (ring should've resized)
	assertSome(t, r.PopBack, 1)
	assertSome(t, r.PopBack, 2)
	assertSome(t, r.PopBack, 3)
	assertSome(t, r.PopBack, 4)
	assertSome(t, r.PopBack, 5)
	assertNone(t, r.PopBack)

	assertLength(t, &r, 0)
}

func BenchmarkFIFO_Ring(b *testing.B) {
	b.ReportAllocs()
	ring := MakeRing[int](0, 16)

	assert := func(expected int) {
		v, ok := ring.PopFront()
		if !ok {
			b.Error("expected value on ring")
		}
		if v != expected {
			b.Error("expected", expected, "but got", v)
		}
	}

	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			ring.PushBack(j)
		}

		for j := 0; j < 10; j++ {
			assert(j)
		}
	}
}

func BenchmarkFIFO_StdRing(b *testing.B) {
	b.ReportAllocs()
	ring := ring2.New(16)
	start := ring

	assert := func(expected int) {
		if start.Value != expected {
			b.Error("expected", expected, "but got", start.Value)
		}
		start = start.Next()
	}

	for i := 0; i < b.N; i++ {
		for j := 0; j < 10; j++ {
			ring.Value = j
			ring = ring.Next()
		}

		for j := 0; j < 10; j++ {
			assert(j)
		}
	}
}

func BenchmarkFIFO_Slice(b *testing.B) {
	b.ReportAllocs()
	slice := make([]int, 0, 16)

	assert := func(expected int) {
		if len(slice) == 0 {
			b.Error("expected value on slice")
		}
		v := slice[0]
		if v != expected {
			b.Error("expected", expected, "but got", v)
		}
		copy(slice, slice[1:])
		slice = slice[:len(slice)-1]
	}

	for i := 0; i < b.N; i++ {
		// pushing is easy for slices
		for j := 0; j < 10; j++ {
			slice = append(slice, j)
		}

		// popping is a bit more complicated
		for j := 0; j < 10; j++ {
			assert(j)
		}
	}
}

func BenchmarkFIFO_Slice2(b *testing.B) {
	b.ReportAllocs()
	slice := make([]int, 0, 16)

	assert := func(expected int) {
		if len(slice) == 0 {
			b.Error("expected value on slice")
		}
		v := slice[0]
		if v != expected {
			b.Error("expected", expected, "but got", v)
		}
		slice = slice[1:]
	}

	for i := 0; i < b.N; i++ {
		// pushing is easy for slices
		for j := 0; j < 10; j++ {
			slice = append(slice, j)
		}

		// popping is a bit more complicated
		for j := 0; j < 10; j++ {
			assert(j)
		}
	}
}

func BenchmarkFIFO_Channel(b *testing.B) {
	b.ReportAllocs()
	channel := make(chan int, 16)

	assert := func(expected int) {
		v, ok := <-channel
		if !ok {
			b.Error("expected value on channel")
		}
		if v != expected {
			b.Error("expected", expected, "but got", v)
		}
	}

	for i := 0; i < b.N; i++ {
		// channel is the easiest interface by far
		for j := 0; j < 10; j++ {
			channel <- j
		}

		for j := 0; j < 10; j++ {
			assert(j)
		}
	}
}
