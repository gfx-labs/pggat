package session

import (
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/gat"
	"pggat2/lib/util/chans"
	"pggat2/lib/zap"
)

type Pool struct {
	// use slice lifo for better perf
	queue []uuid.UUID
	conns map[uuid.UUID]zap.ReadWriter
	qmu   sync.RWMutex

	signal chan struct{}
}

func NewPool() *Pool {
	return &Pool{
		signal: make(chan struct{}),
	}
}

func (T *Pool) acquire(ctx *gat.Context) (uuid.UUID, zap.ReadWriter) {
	for {
		T.qmu.Lock()
		if len(T.queue) > 0 {
			id := T.queue[len(T.queue)-1]
			T.queue = T.queue[:len(T.queue)-1]
			conn, ok := T.conns[id]
			T.qmu.Unlock()
			if !ok {
				continue
			}
			return id, conn
		}
		T.qmu.Unlock()
		if ctx.OnWait != nil {
			chans.TrySend(ctx.OnWait, struct{}{})
		}
		<-T.signal
	}
}

func (T *Pool) release(id uuid.UUID) {
	T.qmu.Lock()
	defer T.qmu.Unlock()
	T.queue = append(T.queue, id)

	chans.TrySend(T.signal, struct{}{})
}

func (T *Pool) Serve(ctx *gat.Context, client zap.ReadWriter) {
	id, server := T.acquire(ctx)
	for {
		clientErr, serverErr := bouncers.Bounce(client, server)
		if clientErr != nil || serverErr != nil {
			_ = client.Close()
			if serverErr == nil {
				T.release(id)
			} else {
				_ = server.Close()
				T.qmu.Lock()
				delete(T.conns, id)
				T.qmu.Unlock()
			}
			break
		}
	}
}

func (T *Pool) AddServer(server zap.ReadWriter) uuid.UUID {
	T.qmu.Lock()
	defer T.qmu.Unlock()

	id := uuid.New()
	if T.conns == nil {
		T.conns = make(map[uuid.UUID]zap.ReadWriter)
	}
	T.conns[id] = server
	T.queue = append(T.queue, id)
	return id
}

func (T *Pool) GetServer(id uuid.UUID) zap.ReadWriter {
	T.qmu.Lock()
	defer T.qmu.Unlock()

	return T.conns[id]
}

func (T *Pool) RemoveServer(id uuid.UUID) zap.ReadWriter {
	T.qmu.Lock()
	defer T.qmu.Unlock()

	conn, ok := T.conns[id]
	if !ok {
		return nil
	}
	delete(T.conns, id)
	return conn
}

func (T *Pool) ReadMetrics(metrics *Metrics) {
	// TODO(garet) metrics
}

var _ gat.RawPool = (*Pool)(nil)
