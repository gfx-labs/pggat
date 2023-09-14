package ring

import (
	ring2 "container/ring"
	"testing"
)

func assertSome[T comparable](t *testing.T, f func() (T, bool), value T) {
	v, ok := f()
	if !ok {
		t.Error("expected items but go nothing")
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

func TestRing_Clear(t *testing.T) {
	var r Ring[int]
	r.PushFront(1)
	r.PushBack(2)
	r.PushFront(3)
	r.PushBack(4)

	assertLength(t, &r, 4)

	r.Clear()

	assertLength(t, &r, 0)

	assertNone(t, r.PopFront)
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
		ring.PushBack(1)
		ring.PushBack(2)
		ring.PushBack(3)
		ring.PushBack(4)
		ring.PushBack(5)
		ring.PushBack(6)
		ring.PushBack(7)
		ring.PushBack(8)
		ring.PushBack(9)
		ring.PushBack(10)

		assert(1)
		assert(2)
		assert(3)
		assert(4)
		assert(5)
		assert(6)
		assert(7)
		assert(8)
		assert(9)
		assert(10)
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
		ring.Value = 1
		ring = ring.Next()
		ring.Value = 2
		ring = ring.Next()
		ring.Value = 3
		ring = ring.Next()
		ring.Value = 4
		ring = ring.Next()
		ring.Value = 5
		ring = ring.Next()
		ring.Value = 6
		ring = ring.Next()
		ring.Value = 7
		ring = ring.Next()
		ring.Value = 8
		ring = ring.Next()
		ring.Value = 9
		ring = ring.Next()
		ring.Value = 10
		ring = ring.Next()

		assert(1)
		assert(2)
		assert(3)
		assert(4)
		assert(5)
		assert(6)
		assert(7)
		assert(8)
		assert(9)
		assert(10)
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
		slice = append(slice, 1)
		slice = append(slice, 2)
		slice = append(slice, 3)
		slice = append(slice, 4)
		slice = append(slice, 5)
		slice = append(slice, 6)
		slice = append(slice, 7)
		slice = append(slice, 8)
		slice = append(slice, 9)
		slice = append(slice, 10)

		// popping is a bit more complicated
		assert(1)
		assert(2)
		assert(3)
		assert(4)
		assert(5)
		assert(6)
		assert(7)
		assert(8)
		assert(9)
		assert(10)
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
		slice = append(slice, 1)
		slice = append(slice, 2)
		slice = append(slice, 3)
		slice = append(slice, 4)
		slice = append(slice, 5)
		slice = append(slice, 6)
		slice = append(slice, 7)
		slice = append(slice, 8)
		slice = append(slice, 9)
		slice = append(slice, 10)

		// popping is a bit more complicated
		assert(1)
		assert(2)
		assert(3)
		assert(4)
		assert(5)
		assert(6)
		assert(7)
		assert(8)
		assert(9)
		assert(10)
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
		channel <- 1
		channel <- 2
		channel <- 3
		channel <- 4
		channel <- 5
		channel <- 6
		channel <- 7
		channel <- 8
		channel <- 9
		channel <- 10

		assert(1)
		assert(2)
		assert(3)
		assert(4)
		assert(5)
		assert(6)
		assert(7)
		assert(8)
		assert(9)
		assert(10)
	}
}
