package shard

import (
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/metrics"
	"math/rand"
	"sync"
	"time"
)

type Shard struct {
	primary  Pool[*conn]
	replicas []Pool[*conn]

	pool *config.Pool
	user *config.User
	conf *config.Shard

	database gat.Database

	options []protocol.FieldsStartupMessageParameters

	dialer gat.Dialer

	mu sync.RWMutex
}

func FromConfig(dialer gat.Dialer, options []protocol.FieldsStartupMessageParameters, pool *config.Pool, user *config.User, conf *config.Shard, database gat.Database) *Shard {
	out := &Shard{
		pool: pool,
		user: user,
		conf: conf,

		database: database,

		options: options,

		dialer: dialer,
	}
	out.init()
	return out
}

func (s *Shard) newConn(conf *config.Server, replicaId int) *conn {
	return &conn{
		conf:      conf,
		replicaId: replicaId,
		s:         s,
	}
}

func (s *Shard) init() {
	s.mu.Lock()
	defer s.mu.Unlock()
	poolSize := s.user.PoolSize
	for _, serv := range s.conf.Servers {
		pool := NewChannelPool[*conn](poolSize)
		for i := 0; i < poolSize; i++ {
			pool.Put(s.newConn(serv, len(s.replicas)))
		}
		switch serv.Role {
		case config.SERVERROLE_PRIMARY:
			s.primary = pool
		case config.SERVERROLE_REPLICA:
			s.replicas = append(s.replicas, pool)
		}
	}
}

func (s *Shard) tryAcquireAvailableReplica() *conn {
	// try to get any available conn
	for _, replica := range s.replicas {
		c, ok := replica.TryGet()
		if ok {
			return c.acquire()
		}
	}

	return nil
}

func (s *Shard) Choose(role config.ServerRole) *conn {
	s.mu.RLock()
	defer s.mu.RUnlock()
	start := time.Now()
	defer func() {
		metrics.RecordWaitTime(s.database.GetName(), s.user.Name, time.Since(start))
	}()
	switch role {
	case config.SERVERROLE_PRIMARY:
		return s.primary.Get().acquire()
	case config.SERVERROLE_REPLICA:
		if len(s.replicas) == 0 {
			// only return primary if primary reads are enabled
			if s.pool.PrimaryReadsEnabled {
				return s.primary.Get().acquire()
			}
			return nil
		}

		// read from a random replica
		acq := s.tryAcquireAvailableReplica()
		if acq != nil {
			return acq
		}

		// try to fall back to primary if there is available resources there
		if s.pool.PrimaryReadsEnabled {
			c, ok := s.primary.TryGet()
			if ok {
				return c.acquire()
			}
		}

		// wait on a random conn
		return s.replicas[rand.Intn(len(s.replicas))].Get().acquire()
	default:
		return nil
	}
}

func (s *Shard) Return(conn *conn) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	switch conn.conf.Role {
	case config.SERVERROLE_PRIMARY:
		s.primary.Put(conn)
	case config.SERVERROLE_REPLICA:
		s.replicas[conn.replicaId].Put(conn)
	default:
	}
}
