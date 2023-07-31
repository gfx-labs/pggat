package session

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/gat"
	"pggat2/lib/util/chans"
	"pggat2/lib/util/maps"
	"pggat2/lib/util/ring"
	"pggat2/lib/zap"
)

type queueItem struct {
	added time.Time
	id    uuid.UUID
}

type Pool struct {
	// use slice lifo for better perf
	queue ring.Ring[queueItem]
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
	for T.queue.Length() == 0 {
		chans.TrySend(ctx.OnWait, struct{}{})
		T.ready.Wait()
	}

	entry, _ := T.queue.PopBack()
	return entry.id, T.conns[entry.id]
}

func (T *Pool) _release(id uuid.UUID) {
	T.queue.PushBack(queueItem{
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

func (T *Pool) ScaleDown(amount int) (remaining int) {
	remaining = amount

	T.qmu.Lock()
	defer T.qmu.Unlock()

	for i := 0; i < amount; i++ {
		v, ok := T.queue.PopFront()
		if !ok {
			break
		}

		conn, ok := T.conns[v.id]
		if !ok {
			continue
		}
		delete(T.conns, v.id)

		_ = conn.Close()
		remaining--
	}

	return
}

func (T *Pool) IdleSince() time.Time {
	T.qmu.Lock()
	defer T.qmu.Unlock()

	v, _ := T.queue.Get(0)
	return v.added
}

func (T *Pool) ReadMetrics(metrics *Metrics) {
	maps.Clear(metrics.Workers)

	if metrics.Workers == nil {
		metrics.Workers = make(map[uuid.UUID]WorkerMetrics)
	}

	T.qmu.Lock()
	defer T.qmu.Unlock()

	for i := 0; i < T.queue.Length(); i++ {
		item, _ := T.queue.Get(i)
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
