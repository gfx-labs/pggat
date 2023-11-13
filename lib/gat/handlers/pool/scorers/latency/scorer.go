package latency

import (
	"time"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
)

func init() {
	caddy.RegisterModule((*Scorer)(nil))
}

type Scorer struct {
	Threshold caddy.Duration `json:"threshold"`
}

func (T *Scorer) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.pool.scorers.latency",
		New: func() caddy.Module {
			return new(Scorer)
		},
	}
}

func (T *Scorer) Score(conn *fed.Conn) (int, error) {
	start := time.Now()
	err, _ := backends.QueryString(conn, nil, "select 0")
	if err != nil {
		return 0, err
	}
	dur := time.Since(start)
	penalty := int(dur / time.Duration(T.Threshold))
	return penalty, nil
}

var _ pool.Scorer = (*Scorer)(nil)
var _ caddy.Module = (*Scorer)(nil)
