package session

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/gat"
	"pggat2/lib/util/chans"
	"pggat2/lib/util/maps"
	"pggat2/lib/util/ring"
	"pggat2/lib/util/strings"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type queueItem struct {
	added time.Time
	id    uuid.UUID
}

type Pool struct {
	roundRobin bool

	// use slice lifo for better perf
	queue ring.Ring[queueItem]
	conns map[uuid.UUID]Conn
	ready sync.Cond
	qmu   sync.Mutex
}

// NewPool creates a new session pool.
// roundRobin determines which order connections will be chosen. If roundRobin = false, connections are handled lifo,
// otherwise they are chosen fifo
func NewPool(roundRobin bool) *Pool {
	p := &Pool{
		roundRobin: roundRobin,
	}
	p.ready.L = &p.qmu
	return p
}

func (T *Pool) acquire(ctx *gat.Context) Conn {
	T.qmu.Lock()
	defer T.qmu.Unlock()
	for T.queue.Length() == 0 {
		chans.TrySend(ctx.OnWait, struct{}{})
		T.ready.Wait()
	}

	var entry queueItem
	if T.roundRobin {
		entry, _ = T.queue.PopFront()
	} else {
		entry, _ = T.queue.PopBack()
	}
	return T.conns[entry.id]
}

func (T *Pool) _release(id uuid.UUID) {
	T.queue.PushBack(queueItem{
		added: time.Now(),
		id:    id,
	})

	T.ready.Signal()
}

func (T *Pool) close(conn Conn) {
	_ = conn.rw.Close()
	T.qmu.Lock()
	defer T.qmu.Unlock()

	delete(T.conns, conn.id)
}

func (T *Pool) release(conn Conn) {
	// reset session state
	err := backends.Query(conn.rw, "DISCARD ALL")
	if err != nil {
		T.close(conn)
		return
	}

	T.qmu.Lock()
	defer T.qmu.Unlock()
	T._release(conn.id)
}

func (T *Pool) Serve(ctx *gat.Context, client zap.ReadWriter, startupParameters map[string]string) {
	defer func() {
		_ = client.Close()

	}()

	connOk := true
	conn := T.acquire(ctx)
	defer func() {
		if connOk {
			T.release(conn)
		} else {
			T.close(conn)
		}
	}()

	if func() bool {
		pkts := zap.NewPackets()
		defer pkts.Done()
		for key, value := range conn.initialParameters {
			if _, ok := startupParameters[key]; ok {
				continue
			}
			packet := zap.NewPacket()
			packets.WriteParameterStatus(packet, key, value)
			pkts.Append(packet)
		}

		for key, value := range startupParameters {
			err := backends.Query(conn.rw, "SET "+key+" = '"+strings.Escape(value, "'")+"'")
			if err != nil {
				connOk = false
				return true
			}
			packet := zap.NewPacket()
			packets.WriteParameterStatus(packet, key, value)
			pkts.Append(packet)
		}

		err := client.WriteV(pkts)
		if err != nil {
			return true
		}
		return false
	}() {
		return
	}

	for {
		clientErr, serverErr := bouncers.Bounce(client, conn.rw)
		if clientErr != nil || serverErr != nil {
			connOk = serverErr == nil
			break
		}
	}
}

func (T *Pool) AddServer(server zap.ReadWriter, parameters map[string]string) uuid.UUID {
	T.qmu.Lock()
	defer T.qmu.Unlock()

	id := uuid.New()
	if T.conns == nil {
		T.conns = make(map[uuid.UUID]Conn)
	}
	T.conns[id] = Conn{
		id:                id,
		rw:                server,
		initialParameters: parameters,
	}
	T._release(id)
	return id
}

func (T *Pool) GetServer(id uuid.UUID) zap.ReadWriter {
	T.qmu.Lock()
	defer T.qmu.Unlock()

	return T.conns[id].rw
}

func (T *Pool) RemoveServer(id uuid.UUID) zap.ReadWriter {
	T.qmu.Lock()
	defer T.qmu.Unlock()

	conn, ok := T.conns[id]
	if !ok {
		return nil
	}
	delete(T.conns, id)
	return conn.rw
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

		_ = conn.rw.Close()
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
