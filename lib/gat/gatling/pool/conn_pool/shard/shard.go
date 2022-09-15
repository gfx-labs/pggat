package shard

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/pool/conn_pool/shard/server"
	"math/rand"
	"sync"
)

type Shard struct {
	primary  gat.Connection
	replicas []gat.Connection

	mu sync.Mutex
}

func FromConfig(user *config.User, conf *config.Shard) *Shard {
	out := &Shard{}
	for _, s := range conf.Servers {
		srv, err := server.Dial(context.TODO(), s.Host, s.Port, user, conf.Database, s.Username, s.Password)
		if err != nil {
			continue
		}
		switch s.Role {
		case config.SERVERROLE_PRIMARY:
			out.primary = srv
		default:
			out.replicas = append(out.replicas, srv)
		}
	}
	return out
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

var _ gat.Shard = (*Shard)(nil)
