package prom

import (
	"gfx.cafe/open/gotoprom"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	gotoprom.MustInit(&PoolHybrid, "pggat_pool_hybrid", prometheus.Labels{})
	gotoprom.MustInit(&OperationHybrid, "pggat_operation_hybrid", prometheus.Labels{})
}

type PoolHybridLabels struct {
	Mode string `label:"hybrid_mode"`
}

type OperationHybridLabels struct {
	Pool string `label:"pool"`
	Mode string `label:"mode"`

	Target string `label:"target"`
}

func (s *PoolHybridLabels) ToOperation(
	target string,
) OperationHybridLabels {
	return OperationHybridLabels{
		Pool:   "hybrid",
		Mode:   s.Mode,
		Target: target,
	}
}

var PoolHybrid struct {
	Accepted func(PoolHybridLabels) prometheus.Counter `name:"accepted" help:"hybrid connections accepted"`
	Current  func(PoolHybridLabels) prometheus.Gauge   `name:"current" help:"current hybrid connections"`
}

var OperationHybrid struct {
	Acquire   func(OperationHybridLabels) prometheus.Histogram `name:"acquire_ms"    buckets:"0.005,0.01,0.1,0.25,0.5,0.75,1,5,10,100,500,1000,5000"  help:"ms to acquire from pool"`
	Execution func(OperationHybridLabels) prometheus.Histogram `name:"execution_ms"  buckets:"1,5,10,30,75,150,300,500,1000,2000,5000,7500,10000,15000,30000" help:"ms that the txn took to execute on remote"`
	Miss      func(OperationHybridLabels) prometheus.Counter   `name:"write_misses" help:"queries which failed replica"`
	Hit       func(OperationHybridLabels) prometheus.Counter   `name:"write_hits" help:"queries which failed replica"`
}
