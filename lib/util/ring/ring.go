package ring

type Ring[T any] struct {
	buffer []T
	head   int
	tail   int
	length int
}

func MakeRing[T any](length, capacity int) Ring[T] {
	if length > capacity {
		// programmer error, panic
		panic("length must be < capacity")
	}
	if capacity < 0 {
		panic("capacity must be >= 0")
	}
	tail := length + 1
	if tail >= capacity {
		tail -= capacity
	}
	return Ring[T]{
		buffer: make([]T, capacity),
		head:   0,
		length: length,
		tail:   tail,
	}
}

func NewRing[T any](length, capacity int) *Ring[T] {
	ring := MakeRing[T](length, capacity)
	return &ring
}

func (r *Ring[T]) grow() {
	if cap(r.buffer) == 0 {
		// special case, uninitialized
		r.buffer = make([]T, 2)
		r.head = 0
		r.tail = 1
		return
	}

	// make new buffer with twice as much space
	buf := make([]T, cap(r.buffer)*2)

	// copy from [head, end of buffer] into new buffer
	copy(buf, r.buffer[r.head:])
	// copy from [beginning of buffer, tail) into new buffer
	copy(buf[len(r.buffer)-r.head:], r.buffer[:r.tail])

	r.tail = len(r.buffer) - r.head + r.tail
	r.head = 0
	r.buffer = buf
}

func (r *Ring[T]) PushBack(value T) {
	if r == nil {
		panic("PushBack() on nil Ring")
	}
	if r.length == cap(r.buffer) {
		r.grow()
	}
	r.buffer[r.tail] = value
	r.tail++
	if r.tail >= len(r.buffer) {
		r.tail -= len(r.buffer)
	}
	r.length++
}

func (r *Ring[T]) PushFront(value T) {
	if r == nil {
		panic("PushFront() on nil Ring")
	}
	if r.length == cap(r.buffer) {
		r.grow()
	}
	r.buffer[r.head] = value
	r.head--
	if r.head < 0 {
		r.head += len(r.buffer)
	}
	r.length++
}

func (r *Ring[T]) PopBack() (T, bool) {
	if r == nil || r.length == 0 {
		return *new(T), false
	}
	tail := r.tail - 1
	if tail < 0 {
		tail += len(r.buffer)
	}
	r.tail = tail
	r.length--
	return r.buffer[tail], true
}

func (r *Ring[T]) PopFront() (T, bool) {
	if r == nil || r.length == 0 {
		return *new(T), false
	}
	head := r.head + 1
	if head >= len(r.buffer) {
		head -= len(r.buffer)
	}
	r.head = head
	r.length--
	return r.buffer[head], true
}

func (r *Ring[T]) Get(i int) (T, bool) {
	if r == nil || i >= r.length || i < 0 {
		return *new(T), false
	}
	ptr := r.head + 1 + i
	if ptr >= len(r.buffer) {
		ptr -= len(r.buffer)
	}
	return r.buffer[ptr], true
}

func (r *Ring[T]) Length() int {
	if r == nil {
		return 0
	}
	return r.length
}

func (r *Ring[T]) Capacity() int {
	if r == nil {
		return 0
	}
	return cap(r.buffer)
}
