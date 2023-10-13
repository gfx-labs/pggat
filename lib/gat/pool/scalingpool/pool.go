package scalingpool

import (
	"sync/atomic"
	"time"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/pool/recipepool"

	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
)

const pendingScaleUpSize = 4

type Pool struct {
	config Config

	servers recipepool.Pool

	started atomic.Bool

	scale  chan struct{}
	closed chan struct{}
}

func MakePool(config Config) Pool {
	return Pool{
		config: config,

		servers: recipepool.MakePool(config.Config),

		scale:  make(chan struct{}, pendingScaleUpSize),
		closed: make(chan struct{}),
	}
}

func (T *Pool) init() {
	if !T.started.Swap(true) {
		go T.scaleLoop()
	}
}

func (T *Pool) AddClient(client *pool.Conn) {
	T.servers.AddClient(client)
}

func (T *Pool) RemoveClient(client *pool.Conn) {
	T.servers.RemoveClient(client)
}

func (T *Pool) AddRecipe(name string, r *pool.Recipe) {
	T.init()

	T.servers.AddRecipe(name, r)
}

func (T *Pool) RemoveRecipe(name string) {
	T.servers.RemoveRecipe(name)
}

func (T *Pool) RemoveServer(server *pool.Conn) {
	T.servers.RemoveServer(server)
}

func (T *Pool) Acquire(client *pool.Conn) (server *pool.Conn) {
	server = T.servers.Acquire(client, gat.SyncModeNonBlocking)
	if server == nil {
		select {
		case T.scale <- struct{}{}:
		default:
		}

		server = T.servers.Acquire(client, gat.SyncModeBlocking)
	}

	return
}

func (T *Pool) Release(server *pool.Conn) {
	T.servers.Release(server)
}

func (T *Pool) Cancel(server *pool.Conn) {
	T.servers.Cancel(server)
}

func (T *Pool) ReadMetrics(m *metrics.Pool) {
	T.servers.ReadMetrics(m)
}

func (T *Pool) Close() {
	close(T.closed)

	T.servers.Close()
}

func (T *Pool) scaleLoop() {
	var idle *time.Timer
	if T.config.ServerIdleTimeout != 0 {
		idle = time.NewTimer(T.config.ServerIdleTimeout)
	}

	var backoff time.Duration
	var scale *time.Timer

	for {
		var idle1 <-chan time.Time
		if idle != nil {
			idle1 = idle.C
		}

		var scale1 <-chan struct{}
		var scale2 <-chan time.Time
		if backoff != 0 {
			scale1 = T.scale
		} else {
			scale2 = scale.C
		}

		select {
		case <-idle1:
			idle.Reset(T.servers.ScaleDown(T.config.ServerIdleTimeout))
		case <-scale1:
			if !T.servers.ScaleUp() {
				backoff = T.config.ServerReconnectInitialTime
				if backoff == 0 {
					continue
				}
				if scale == nil {
					scale = time.NewTimer(backoff)
				} else {
					scale.Reset(backoff)
				}
				continue
			}
		case <-scale2:
			if !T.servers.ScaleUp() {
				backoff *= 2
				if T.config.ServerReconnectMaxTime != 0 && backoff > T.config.ServerReconnectMaxTime {
					backoff = T.config.ServerReconnectMaxTime
				}
				scale.Reset(backoff)
				continue
			}
			backoff = 0
		case <-T.closed:
			return
		}
	}
}
