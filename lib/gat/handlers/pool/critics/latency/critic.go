package latency

import (
	"context"
	"time"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
)

func init() {
	caddy.RegisterModule((*Critic)(nil))
}

type Critic struct {
	Threshold caddy.Duration `json:"threshold"`
	Validity  caddy.Duration `json:"validity"`
}

func (T *Critic) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.pool.critics.latency",
		New: func() caddy.Module {
			return new(Critic)
		},
	}
}

func (T *Critic) Taste(ctx context.Context, conn *fed.Conn) (int, time.Duration, error) {
	start := time.Now()
	err, _ := backends.QueryString(ctx, conn, nil, "select 0")
	if err != nil {
		return 0, time.Duration(T.Validity), err
	}
	dur := time.Since(start)
	penalty := int(dur / time.Duration(T.Threshold))
	return penalty, time.Duration(T.Validity), nil
}

var _ pool.Critic = (*Critic)(nil)
var _ caddy.Module = (*Critic)(nil)
