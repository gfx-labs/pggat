package serverpool

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/gat/pool2"
)

type Pool struct {
	config Config
	pooler pool.Pooler

	servers map[uuid.UUID]*pool2.Conn
	mu      sync.RWMutex
}

func MakePool(config Config) Pool {
	return Pool{
		config: config,
		pooler: config.NewPooler(),
	}
}

func (T *Pool) AddClient(client *pool2.Conn) {
	T.pooler.AddClient(client.ID)
}

func (T *Pool) RemoveClient(client *pool2.Conn) {
	T.pooler.DeleteClient(client.ID)
}

func (T *Pool) AddServer(server *pool2.Conn) {
	func() {
		T.mu.Lock()
		defer T.mu.Unlock()
		if T.servers == nil {
			T.servers = make(map[uuid.UUID]*pool2.Conn)
		}
		T.servers[server.ID] = server
	}()
	T.pooler.AddServer(server.ID)
}

func (T *Pool) RemoveServer(server *pool2.Conn) {
	T.pooler.DeleteServer(server.ID)

	T.mu.Lock()
	defer T.mu.Unlock()
	delete(T.servers, server.ID)
}

func (T *Pool) Acquire(client *pool2.Conn, mode pool.SyncMode) (server *pool2.Conn) {
	for {
		serverID := T.pooler.Acquire(client.ID, mode)
		if serverID == uuid.Nil {
			return
		}

		T.mu.RLock()
		server, _ = T.servers[serverID]
		T.mu.RUnlock()

		if server != nil {
			return
		}

		T.pooler.DeleteServer(serverID)
	}
}

func (T *Pool) Release(server *pool2.Conn) {
	if T.config.ServerResetQuery != "" {
		pool2.SetConnState(metrics.ConnStateRunningResetQuery, server)

		err, _ := backends.QueryString(server.Conn, nil, T.config.ServerResetQuery)
		if err != nil {
			T.RemoveServer(server)
			return
		}
	}

	pool2.SetConnState(metrics.ConnStateIdle, server)

	T.pooler.Release(server.ID)
}

// ScaleDown removes any servers that have been idle for longer than idleFor. Returns the next time to attempt to scale
// down again
func (T *Pool) ScaleDown(idleFor time.Duration) time.Duration {
	T.mu.Lock()
	defer T.mu.Unlock()

	now := time.Now()

	var oldest time.Time

	for id, server := range T.servers {
		state, since := server.GetState()
		if state != metrics.ConnStateIdle {
			continue
		}

		dur := now.Sub(since)
		if dur > idleFor {
			T.pooler.DeleteServer(id)
			delete(T.servers, id)
		}

		if oldest != (time.Time{}) && since.Before(oldest) {
			oldest = since
		}
	}

	dur := now.Sub(oldest)
	if dur > idleFor {
		dur = idleFor
	}

	return dur
}

func (T *Pool) ReadMetrics(m *metrics.Pool) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	if m.Servers == nil {
		m.Servers = make(map[uuid.UUID]metrics.Conn)
	}

	for id, server := range T.servers {
		var c metrics.Conn
		server.ReadMetrics(&c)
		m.Servers[id] = c
	}
}

func (T *Pool) Close() {
	T.mu.Lock()
	defer T.mu.Unlock()

	for id, server := range T.servers {
		_ = server.Conn.Close()
		delete(T.servers, id)
	}
}
