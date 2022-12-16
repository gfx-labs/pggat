package shard

import (
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"gfx.cafe/gfx/pggat/lib/metrics"
	"math/rand"
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
	poolSize := s.user.PoolSize
	for _, serv := range s.conf.Servers {
		pool := NewChannelPool[*conn](poolSize)
		for i := 0; i < poolSize; i++ {
			pool.Put(s.newConn(serv, len(s.replicas)))
		}
		switch serv.Role {
		case config.SERVERROLE_PRIMARY:
			s.primary = pool
		default:
			s.replicas = append(s.replicas, pool)
		}
	}
}

func (s *Shard) Choose(role config.ServerRole) *conn {
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
		return s.replicas[rand.Intn(len(s.replicas))].Get().acquire()
	default:
		return nil
	}
}

func (s *Shard) Return(conn *conn) {
	switch conn.conf.Role {
	case config.SERVERROLE_PRIMARY:
		s.primary.Put(conn)
	case config.SERVERROLE_REPLICA:
		s.replicas[conn.replicaId].Put(conn)
	default:
	}
}

func (s *Shard) GetPrimary() gat.Connection {
	return s.Choose(config.SERVERROLE_PRIMARY)
}

func (s *Shard) GetReplica() gat.Connection {
	return s.Choose(config.SERVERROLE_REPLICA)
}
