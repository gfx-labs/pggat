package session

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/bouncer"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/gat"
	"pggat2/lib/util/chans"
	"pggat2/lib/util/maps"
	"pggat2/lib/util/ring"
	"pggat2/lib/util/slices"
	"pggat2/lib/util/strutil"
	packets "pggat2/lib/zap/packets/v3.0"
)

type queueItem struct {
	added time.Time
	id    uuid.UUID
}

type Pool struct {
	config Config

	// use slice lifo for better perf
	queue ring.Ring[queueItem]
	conns map[uuid.UUID]bouncer.Conn
	ready sync.Cond
	qmu   sync.Mutex
}

// NewPool creates a new session pool.
func NewPool(config Config) *Pool {
	p := &Pool{
		config: config,
	}
	p.ready.L = &p.qmu
	return p
}

func (T *Pool) acquire(ctx *gat.Context) (uuid.UUID, bouncer.Conn) {
	T.qmu.Lock()
	defer T.qmu.Unlock()
	for T.queue.Length() == 0 {
		chans.TrySend(ctx.OnWait, struct{}{})
		T.ready.Wait()
	}

	var entry queueItem
	if T.config.RoundRobin {
		entry, _ = T.queue.PopFront()
	} else {
		entry, _ = T.queue.PopBack()
	}
	return entry.id, T.conns[entry.id]
}

func (T *Pool) _release(id uuid.UUID) {
	T.queue.PushBack(queueItem{
		added: time.Now(),
		id:    id,
	})

	T.ready.Signal()
}

func (T *Pool) close(id uuid.UUID, conn bouncer.Conn) {
	_ = conn.RW.Close()
	T.qmu.Lock()
	defer T.qmu.Unlock()

	delete(T.conns, id)
}

func (T *Pool) release(id uuid.UUID, conn bouncer.Conn) {
	// reset session state
	err := backends.QueryString(&backends.Context{}, conn.RW, "DISCARD ALL")
	if err != nil {
		T.close(id, conn)
		return
	}

	T.qmu.Lock()
	defer T.qmu.Unlock()
	T._release(id)
}

func (T *Pool) Serve(ctx *gat.Context, client bouncer.Conn) {
	defer func() {
		_ = client.RW.Close()
	}()

	serverOK := true
	serverID, server := T.acquire(ctx)
	defer func() {
		if serverOK {
			T.release(serverID, server)
		} else {
			T.close(serverID, server)
		}
	}()

	if func() bool {
		add := func(key strutil.CIString) error {
			if value, ok := server.InitialParameters[key]; ok {
				ps := packets.ParameterStatus{
					Key:   key.String(),
					Value: value,
				}

				if err := client.RW.WritePacket(ps.IntoPacket()); err != nil {
					return err
				}
			}
			return nil
		}

		for key, value := range client.InitialParameters {
			// skip already set params
			if server.InitialParameters[key] == value {
				if err := add(key); err != nil {
					return true
				}
				continue
			}

			// only set tracking params
			if !slices.Contains(T.config.TrackedParameters, key) {
				if err := add(key); err != nil {
					return true
				}
				continue
			}

			ps := packets.ParameterStatus{
				Key:   key.String(),
				Value: value,
			}
			if err := client.RW.WritePacket(ps.IntoPacket()); err != nil {
				return true
			}

			if err := backends.SetParameter(&backends.Context{}, server.RW, key, value); err != nil {
				serverOK = false
				return true
			}
		}

		for key := range server.InitialParameters {
			if _, ok := client.InitialParameters[key]; ok {
				continue
			}

			if err := add(key); err != nil {
				return true
			}
		}

		return false
	}() {
		return
	}

	for {
		packet, err := client.RW.ReadPacket(true)
		if err != nil {
			break
		}
		clientErr, serverErr := bouncers.Bounce(client.RW, server.RW, packet)
		if clientErr != nil || serverErr != nil {
			serverOK = serverErr == nil
			break
		}
	}
}

func (T *Pool) LookupCorresponding(key [8]byte) (uuid.UUID, [8]byte, bool) {
	// TODO(garet)
	return uuid.Nil, [8]byte{}, false
}

func (T *Pool) AddServer(server bouncer.Conn) uuid.UUID {
	T.qmu.Lock()
	defer T.qmu.Unlock()

	id := uuid.New()
	if T.conns == nil {
		T.conns = make(map[uuid.UUID]bouncer.Conn)
	}
	T.conns[id] = server
	T._release(id)
	return id
}

func (T *Pool) GetServer(id uuid.UUID) bouncer.Conn {
	T.qmu.Lock()
	defer T.qmu.Unlock()

	return T.conns[id]
}

func (T *Pool) RemoveServer(id uuid.UUID) bouncer.Conn {
	T.qmu.Lock()
	defer T.qmu.Unlock()

	conn, ok := T.conns[id]
	if !ok {
		return bouncer.Conn{}
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

		_ = conn.RW.Close()
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
