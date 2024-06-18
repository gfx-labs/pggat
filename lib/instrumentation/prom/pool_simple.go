package prom

import (
	"gfx.cafe/open/gotoprom"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	gotoprom.MustInit(&PoolSimple, "pggat_pool_simple", make(prometheus.Labels))
	gotoprom.MustInit(&OperationSimple, "pggat_operation_simple", make(prometheus.Labels))
}

var PoolSimple struct {
	Accepted func(PoolSimpleLabels) prometheus.Counter `name:"accepted" help:"simple connections accepted"`
	Current  func(PoolSimpleLabels) prometheus.Gauge   `name:"current" help:"current simple connections"`
}

type PoolSimpleLabels struct {
	Mode string `label:"mode"`
}

func (s *PoolSimpleLabels) ToOperation() OperationSimpleLabels {
	return OperationSimpleLabels{
		Pool: "basic",
		Mode: s.Mode,
	}
}

type OperationSimpleLabels struct {
	Pool string `label:"pool"`
	Mode string `label:"mode"`
}

var OperationSimple struct {
	Acquire   func(OperationSimpleLabels) prometheus.Histogram `name:"acquire_ms"    buckets:"0.005,0.01,0.1,0.25,0.5,0.75,1,5,10,100,500,1000"  help:"ms to acquire from pool"`
	Execution func(OperationSimpleLabels) prometheus.Histogram `name:"execution_ms"  buckets:"1,5,10,30,75,150,300,500,1000,2000,5000,7500,10000,15000,30000" help:"ms that the txn took to execute on remote"`
}
