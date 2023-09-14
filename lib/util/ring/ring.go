package ring

type Ring[T any] struct {
	buf []T
	// real head is head-1, like this so nil ring is valid
	head   int
	tail   int
	length int
}

func MakeRing[T any](length, capacity int) Ring[T] {
	if length > capacity {
		panic("length must be less than capacity")
	}
	return Ring[T]{
		buf:    make([]T, capacity),
		tail:   length,
		length: length,
	}
}

func NewRing[T any](length, capacity int) *Ring[T] {
	r := MakeRing[T](length, capacity)
	return &r
}

func (r *Ring[T]) grow() {
	size := len(r.buf) * 2
	if size == 0 {
		size = 2
	}

	buf := make([]T, size)
	copy(buf, r.buf[r.head:])
	copy(buf[len(r.buf[r.head:]):], r.buf[:r.head])
	r.head = 0
	r.tail = r.length
	r.buf = buf
}

func (r *Ring[T]) incHead() {
	// resize
	if r.length == 0 {
		panic("smashing detected")
	}
	r.length--

	r.head++
	if r.head == len(r.buf) {
		r.head = 0
	}
}

func (r *Ring[T]) decHead() {
	// resize
	if r.length == len(r.buf) {
		r.grow()
	}
	r.length++

	r.head--
	if r.head == -1 {
		r.head = len(r.buf) - 1
	}
}

func (r *Ring[T]) incTail() {
	// resize
	if r.length == len(r.buf) {
		r.grow()
	}
	r.length++

	r.tail++
	if r.tail == len(r.buf) {
		r.tail = 0
	}
}

func (r *Ring[T]) decTail() {
	// resize
	if r.length == 0 {
		panic("smashing detected")
	}
	r.length--

	r.tail--
	if r.tail == -1 {
		r.tail = len(r.buf) - 1
	}
}

func (r *Ring[T]) tailSub1() int {
	tail := r.tail - 1
	if tail == -1 {
		tail = len(r.buf) - 1
	}
	return tail
}

func (r *Ring[T]) PopFront() (T, bool) {
	if r.length == 0 {
		return *new(T), false
	}

	front := r.buf[r.head]
	r.incHead()
	return front, true
}

func (r *Ring[T]) PopBack() (T, bool) {
	if r.length == 0 {
		return *new(T), false
	}

	r.decTail()
	return r.buf[r.tail], true
}

func (r *Ring[T]) Clear() {
	r.head = 0
	r.tail = 0
	r.length = 0
}

func (r *Ring[T]) PushFront(value T) {
	r.decHead()
	r.buf[r.head] = value
}

func (r *Ring[T]) PushBack(value T) {
	r.incTail()
	r.buf[r.tailSub1()] = value
}

func (r *Ring[T]) Length() int {
	return r.length
}

func (r *Ring[T]) Capacity() int {
	return len(r.buf)
}

func (r *Ring[T]) Get(n int) T {
	if n >= r.length {
		panic("index out of range")
	}
	ptr := r.head + n
	if ptr >= len(r.buf) {
		ptr -= len(r.buf)
	}
	return r.buf[ptr]
}
