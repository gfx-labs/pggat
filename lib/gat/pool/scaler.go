package pool

import (
	"time"

	"tuxpa.in/a/zlog/log"
)

type scaler struct {
	pool *Pool

	backingOff bool
	backoff    time.Duration

	// timers
	idle    *time.Timer
	pending *time.Timer
}

func newScaler(pool *Pool) *scaler {
	s := &scaler{
		pool:    pool,
		backoff: pool.options.ServerReconnectInitialTime,
	}

	if pool.options.ServerIdleTimeout != 0 {
		s.idle = time.NewTimer(pool.options.ServerIdleTimeout)
	}

	return s
}

func (T *scaler) idleTimeout(now time.Time) {
	// idle loop for scaling down
	var wait time.Duration

	var idlest *pooledServer
	var idleStart time.Time
	for idlest, idleStart = T.pool.idlest(); idlest != nil && now.Sub(idleStart) > T.pool.options.ServerIdleTimeout; idlest, idleStart = T.pool.idlest() {
		T.pool.removeServer(idlest)
	}

	if idlest == nil {
		wait = T.pool.options.ServerIdleTimeout
	} else {
		wait = idleStart.Add(T.pool.options.ServerIdleTimeout).Sub(now)
	}

	T.idle.Reset(wait)
}

func (T *scaler) pendingTimeout() {
	if T.backingOff {
		T.backoff *= 2
		if T.pool.options.ServerReconnectMaxTime != 0 && T.backoff > T.pool.options.ServerReconnectMaxTime {
			T.backoff = T.pool.options.ServerReconnectMaxTime
		}
	}

	for T.pool.pendingCount.Load() > 0 {
		// pending loop for scaling up
		if T.pool.scaleUp() {
			// scale up successful, see if we need to scale up more
			T.backoff = T.pool.options.ServerReconnectInitialTime
			T.backingOff = false
			continue
		}

		if T.backoff == 0 {
			// no backoff
			T.backoff = T.pool.options.ServerReconnectInitialTime
			T.backingOff = false
			continue
		}

		T.backingOff = true
		if T.pending == nil {
			T.pending = time.NewTimer(T.backoff)
		} else {
			T.pending.Reset(T.backoff)
		}

		log.Printf("failed to dial server. trying again in %v", T.backoff)

		return
	}
}

func (T *scaler) Run() {
	for {
		var idle <-chan time.Time
		if T.idle != nil {
			idle = T.idle.C
		}

		var pending1 <-chan struct{}
		var pending2 <-chan time.Time
		if T.backingOff {
			pending2 = T.pending.C
		} else {
			pending1 = T.pool.pending
		}

		select {
		case t := <-idle:
			T.idleTimeout(t)
		case <-pending1:
			T.pendingTimeout()
		case <-pending2:
			T.pendingTimeout()
		case <-T.pool.closed:
			return
		}
	}
}
