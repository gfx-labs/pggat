package schedulers

import (
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/util/ring"
)

type Source struct {
	id uuid.UUID

	queue    ring.Ring[any]
	notifier func(*Source)
	mu       sync.Mutex
}

func newSource() *Source {
	return &Source{
		id: uuid.New(),
	}
}

func (T *Source) Schedule(w any) {
	T.mu.Lock()
	T.queue.PushBack(w)
	notifier := T.notifier
	T.mu.Unlock()

	if notifier != nil {
		notifier(T)
	}
}

func (T *Source) pop() (next any, ok, hasMore bool) {
	T.mu.Lock()
	defer T.mu.Unlock()
	next, ok = T.queue.PopFront()
	hasMore = T.queue.Length() != 0
	return
}

func (T *Source) setNotifier(notifier func(*Source)) {
	T.mu.Lock()
	defer T.mu.Unlock()
	T.notifier = notifier
}

var _ rob.Source = (*Source)(nil)
