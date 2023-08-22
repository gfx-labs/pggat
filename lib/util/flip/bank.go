package flip

import "sync"

type Bank struct {
	pending []func() error
	mu      sync.Mutex
}

func (T *Bank) Queue(fn func() error) {
	T.mu.Lock()
	defer T.mu.Unlock()
	T.pending = append(T.pending, fn)
}

func (T *Bank) take() []func() error {
	T.mu.Lock()
	defer T.mu.Unlock()
	v := T.pending
	T.pending = nil
	return v
}

func (T *Bank) give(sz []func() error) {
	T.mu.Lock()
	defer T.mu.Unlock()
	if T.pending == nil {
		T.pending = sz[:0]
	}
}

func (T *Bank) Wait() error {
	batch := T.take()
	defer T.give(batch)

	if len(batch) == 0 {
		return nil
	}

	if len(batch) == 1 {
		return batch[0]()
	}

	ch := make(chan error, len(T.pending))

	for _, pending := range T.pending {
		go func(pending func() error) {
			ch <- pending()
		}(pending)
	}

	for i := 0; i < len(T.pending); i++ {
		if err := <-ch; err != nil {
			return err
		}
	}

	return nil
}
