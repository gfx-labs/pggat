package transaction

type WorkerPool[T any] interface {
	TryGet() (T, bool)
	Get() T
	Put(T)
}

type ChannelPool[T any] struct {
	ch chan T
}

func NewChannelPool[T any](size int) *ChannelPool[T] {
	return &ChannelPool[T]{
		ch: make(chan T, size*10),
	}
}

func (c *ChannelPool[T]) Get() T {
	return <-c.ch
}

func (c *ChannelPool[T]) TryGet() (T, bool) {
	select {
	case out := <-c.ch:
		return out, true
	default:
		return *new(T), false
	}
}

func (c *ChannelPool[T]) Put(t T) {
	c.ch <- t
}
