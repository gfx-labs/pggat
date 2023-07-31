package session

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/gat"
	"pggat2/lib/util/chans"
	"pggat2/lib/util/maps"
	"pggat2/lib/zap"
)

type queueItem struct {
	added time.Time
	id    uuid.UUID
}

type Pool struct {
	// use slice lifo for better perf
	queue []queueItem
	conns map[uuid.UUID]zap.ReadWriter
	ready sync.Cond
	qmu   sync.Mutex
}

func NewPool() *Pool {
	p := &Pool{}
	p.ready.L = &p.qmu
	return p
}

func (T *Pool) acquire(ctx *gat.Context) (uuid.UUID, zap.ReadWriter) {
	T.qmu.Lock()
	defer T.qmu.Unlock()
	for {
		if len(T.queue) > 0 {
			item := T.queue[len(T.queue)-1]
			T.queue = T.queue[:len(T.queue)-1]
			conn, ok := T.conns[item.id]
			if !ok {
				continue
			}
			return item.id, conn
		}
		if ctx.OnWait != nil {
			chans.TrySend(ctx.OnWait, struct{}{})
		}
		T.ready.Wait()
	}
}

func (T *Pool) _release(id uuid.UUID) {
	T.queue = append(T.queue, queueItem{
		added: time.Now(),
		id:    id,
	})

	T.ready.Signal()
}

func (T *Pool) release(id uuid.UUID) {
	T.qmu.Lock()
	defer T.qmu.Unlock()
	T._release(id)
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
	T._release(id)
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
	maps.Clear(metrics.Workers)

	if metrics.Workers == nil {
		metrics.Workers = make(map[uuid.UUID]WorkerMetrics)
	}

	T.qmu.Lock()
	defer T.qmu.Unlock()

	for _, item := range T.queue {
		metrics.Workers[item.id] = WorkerMetrics{
			LastActive: item.added,
		}
	}

	for id := range T.conns {
		if _, ok := metrics.Workers[id]; !ok {
			metrics.Workers[id] = WorkerMetrics{}
		}
	}
}

var _ gat.RawPool = (*Pool)(nil)
