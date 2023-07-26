package session

import (
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/gat"
	"pggat2/lib/zap"
)

type Pool struct {
	// use slice lifo for better perf
	queue []uuid.UUID
	conns map[uuid.UUID]zap.ReadWriter
	mu    sync.Mutex

	signal chan struct{}
}

func NewPool() *Pool {
	return &Pool{
		signal: make(chan struct{}),
	}
}

func (T *Pool) acquire() (uuid.UUID, zap.ReadWriter) {
	for {
		T.mu.Lock()
		if len(T.queue) > 0 {
			id := T.queue[len(T.queue)-1]
			T.queue = T.queue[:len(T.queue)-1]
			conn, ok := T.conns[id]
			T.mu.Unlock()
			if !ok {
				continue
			}
			return id, conn
		}
		T.mu.Unlock()
		<-T.signal
	}
}

func (T *Pool) release(id uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()
	T.queue = append(T.queue, id)

	select {
	case T.signal <- struct{}{}:
	default:
	}
}

func (T *Pool) Serve(client zap.ReadWriter) {
	id, server := T.acquire()
	for {
		clientErr, serverErr := bouncers.Bounce(client, server)
		if clientErr != nil || serverErr != nil {
			_ = client.Close()
			if serverErr == nil {
				T.release(id)
			} else {
				_ = server.Close()
				T.mu.Lock()
				delete(T.conns, id)
				T.mu.Unlock()
			}
			break
		}
	}
}

func (T *Pool) AddServer(server zap.ReadWriter) uuid.UUID {
	T.mu.Lock()
	defer T.mu.Unlock()

	id := uuid.New()
	if T.conns == nil {
		T.conns = make(map[uuid.UUID]zap.ReadWriter)
	}
	T.conns[id] = server
	T.queue = append(T.queue, id)
	return id
}

func (T *Pool) GetServer(id uuid.UUID) zap.ReadWriter {
	T.mu.Lock()
	defer T.mu.Unlock()

	return T.conns[id]
}

func (T *Pool) RemoveServer(id uuid.UUID) zap.ReadWriter {
	T.mu.Lock()
	defer T.mu.Unlock()

	conn, ok := T.conns[id]
	if !ok {
		return nil
	}
	delete(T.conns, id)
	return conn
}

var _ gat.RawPool = (*Pool)(nil)
