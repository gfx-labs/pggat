package shard

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/pool/conn_pool/shard/server"
	"math/rand"
	"reflect"
)

type Shard struct {
	primary  gat.Connection
	replicas []gat.Connection

	user *config.User
	conf *config.Shard
}

func FromConfig(user *config.User, conf *config.Shard) *Shard {
	out := &Shard{
		user: user,
		conf: conf,
	}
	out.init()
	return out
}

func (s *Shard) init() {
	s.primary = nil
	s.replicas = nil
	for _, serv := range s.conf.Servers {
		srv, err := server.Dial(context.TODO(), serv.Host, serv.Port, s.user, s.conf.Database, serv.Username, serv.Password)
		if err != nil {
			continue
		}
		switch serv.Role {
		case config.SERVERROLE_PRIMARY:
			s.primary = srv
		default:
			s.replicas = append(s.replicas, srv)
		}
	}
}

func (s *Shard) Choose(role config.ServerRole) gat.Connection {
	switch role {
	case config.SERVERROLE_PRIMARY:
		return s.primary
	case config.SERVERROLE_REPLICA:
		if len(s.replicas) == 0 {
			return s.primary
		}

		return s.replicas[rand.Intn(len(s.replicas))]
	default:
		return nil
	}
}

func (s *Shard) GetPrimary() gat.Connection {
	return s.primary
}

func (s *Shard) GetReplicas() []gat.Connection {
	return s.replicas
}

func (s *Shard) EnsureConfig(c *config.Shard) {
	if !reflect.DeepEqual(s.conf, c) {
		s.conf = c
		s.init()
	}
}

var _ gat.Shard = (*Shard)(nil)
