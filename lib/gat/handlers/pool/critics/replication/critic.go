package replication

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gsql"
	"github.com/caddyserver/caddy/v2"
	"time"
)

func init() {
	caddy.RegisterModule((*Critic)(nil))
}

// Critic describes a replication critic which measures replication lag,
// with a fallback to query latency when there is no *measurable* lag
type Critic struct {
	QueryThreshold       caddy.Duration `json:"query_threshold"`
	ReplicationThreshold caddy.Duration `json:"replication_threshold"`
	Validity             caddy.Duration `json:"validity"`
}

func NewCritic() *Critic {
	return &Critic{
		QueryThreshold:       caddy.Duration(time.Millisecond * 300),
		ReplicationThreshold: caddy.Duration(time.Second * 3),
		Validity:             caddy.Duration(time.Minute * 5),
	}
}

func (T *Critic) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.pool.critics.replication",
		New: func() caddy.Module {
			return new(Critic)
		},
	}
}

type replicationLagQueryResult struct {
	Lag *float64 `sql:"0"`
}

const replicationLagQuery = `SELECT CASE WHEN pg_last_wal_receive_lsn() = pg_last_wal_replay_lsn() THEN 0 ELSE EXTRACT (epoch from (now() - pg_last_xact_replay_timestamp())) END AS lag;`

func (T *Critic) Taste(ctx context.Context, conn *fed.Conn) (int, time.Duration, error) {
	var result replicationLagQueryResult

	start := time.Now()
	err := gsql.Query(ctx, conn, []any{&result}, replicationLagQuery)
	if err != nil {
		return 0, time.Duration(0), err
	}

	penalty := 0

	if (result.Lag != nil) && (*result.Lag > 0) {
		penalty = int(*result.Lag / time.Duration(T.ReplicationThreshold).Seconds())
	} else {
		dur := time.Since(start)
		penalty = int(dur / time.Duration(T.QueryThreshold))
	}

	return penalty, time.Duration(T.Validity), nil
}

var _ pool.Critic = (*Critic)(nil)
var _ caddy.Module = (*Critic)(nil)
