package shard

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
	"git.tuxpa.in/a/zlog/log"
	"math/rand"
	"reflect"
)

type shardConn struct {
	conn gat.Connection
	conf *config.Server
	s    *Shard
}

func (s *shardConn) connect() {
	if s.s == nil || s.conf == nil {
		return
	}
	if s.conn != nil {
		_ = s.conn.Close()
	}
	var err error
	s.conn, err = s.s.dialer(context.TODO(), s.s.options, s.s.user, s.s.conf, s.conf)
	if err != nil {
		log.Println("error connecting to server:", err)
	}
	return
}

func (s *shardConn) acquire() gat.Connection {
	if s.conn == nil || s.conn.IsCloseNeeded() {
		s.connect()
	}
	return s.conn
}

type Shard struct {
	primary  shardConn
	replicas []shardConn

	pool *config.Pool
	user *config.User
	conf *config.Shard

	options []protocol.FieldsStartupMessageParameters

	dialer gat.Dialer
}

func FromConfig(dialer gat.Dialer, options []protocol.FieldsStartupMessageParameters, pool *config.Pool, user *config.User, conf *config.Shard) *Shard {
	out := &Shard{
		pool: pool,
		user: user,
		conf: conf,

		options: options,

		dialer: dialer,
	}
	out.init()
	return out
}

func (s *Shard) newConn(conf *config.Server) shardConn {
	return shardConn{
		conf: conf,
		s:    s,
	}
}

func (s *Shard) init() {
	s.primary = shardConn{}
	s.replicas = nil
	for _, serv := range s.conf.Servers {
		switch serv.Role {
		case config.SERVERROLE_PRIMARY:
			s.primary = s.newConn(serv)
		default:
			s.replicas = append(s.replicas, s.newConn(serv))
		}
	}
}

func (s *Shard) Choose(role config.ServerRole) gat.Connection {
	switch role {
	case config.SERVERROLE_PRIMARY:
		return s.primary.acquire()
	case config.SERVERROLE_REPLICA:
		if len(s.replicas) == 0 {
			// only return primary if primary reads are enabled
			if s.pool.PrimaryReadsEnabled {
				return s.primary.acquire()
			}
			return nil
		}

		// read from a random replica
		return s.replicas[rand.Intn(len(s.replicas))].acquire()
	default:
		return nil
	}
}

func (s *Shard) GetPrimary() gat.Connection {
	return s.Choose(config.SERVERROLE_PRIMARY)
}

func (s *Shard) GetReplica() gat.Connection {
	return s.Choose(config.SERVERROLE_REPLICA)
}

func (s *Shard) EnsureConfig(c *config.Shard) {
	if !reflect.DeepEqual(s.conf, c) {
		s.conf = c
		s.init()
	}
}
