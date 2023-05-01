package schedulers

import (
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/rob"
	"pggat2/lib/util/ring"
)

type Source struct {
	id uuid.UUID

	queue  ring.Ring[any]
	notify notifier
	mu     sync.Mutex
}

func newSource() *Source {
	return &Source{
		id: uuid.New(),
	}
}

func (T *Source) Schedule(w any, constraints rob.Constraints) {
	T.mu.Lock()
	T.queue.PushBack(w)
	notify := T.notify
	T.mu.Unlock()

	if notify != nil {
		notify.notify(T)
	}
}

func (T *Source) pop() (next any, ok, hasMore bool) {
	T.mu.Lock()
	defer T.mu.Unlock()
	next, ok = T.queue.PopFront()
	hasMore = T.queue.Length() != 0
	return
}

func (T *Source) setNotifier(notify notifier) {
	T.mu.Lock()
	defer T.mu.Unlock()
	T.notify = notify
}

var _ rob.Source = (*Source)(nil)
