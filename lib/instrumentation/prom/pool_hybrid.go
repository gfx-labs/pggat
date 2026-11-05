package prom

import (
	"github.com/prometheus/client_golang/prometheus"
)

type PoolHybridLabels struct {
	Mode     string
	Database string
	User     string
}

type OperationHybridLabels struct {
	Pool     string
	Mode     string
	Target   string
	Database string
	User     string
}

func (s *PoolHybridLabels) ToOperation(
	target string,
) OperationHybridLabels {
	return OperationHybridLabels{
		Pool:     "hybrid",
		Mode:     s.Mode,
		Database: s.Database,
		Target:   target,
	}
}

func (l OperationHybridLabels) values() []string {
	return []string{l.Pool, l.Mode, l.Target, l.Database, l.User}
}

var (
	poolHybridAccepted = newCounterVec("pggat_pool_hybrid", "accepted", "hybrid connections accepted", "mode", "database", "user")
	poolHybridCurrent  = newGaugeVec("pggat_pool_hybrid", "current", "current hybrid connections", "mode", "database", "user")

	opHybridLabels    = []string{"pool", "mode", "target", "database", "user"}
	opHybridAcquire   = newHistogramVec("pggat_operation_hybrid", "acquire_ms", "ms to acquire from pool", acquireBuckets, opHybridLabels...)
	opHybridExecution = newHistogramVec("pggat_operation_hybrid", "execution_ms", "ms that the txn took to execute on remote", executionBuckets, opHybridLabels...)
	opHybridMiss      = newCounterVec("pggat_operation_hybrid", "write_misses", "queries which failed replica", opHybridLabels...)
	opHybridHit       = newCounterVec("pggat_operation_hybrid", "write_hits", "queries which failed replica", opHybridLabels...)
)

var PoolHybrid = struct {
	Accepted func(PoolHybridLabels) prometheus.Counter
	Current  func(PoolHybridLabels) prometheus.Gauge
}{
	Accepted: func(l PoolHybridLabels) prometheus.Counter {
		return poolHybridAccepted.WithLabelValues(l.Mode, l.Database, l.User)
	},
	Current: func(l PoolHybridLabels) prometheus.Gauge {
		return poolHybridCurrent.WithLabelValues(l.Mode, l.Database, l.User)
	},
}

var OperationHybrid = struct {
	Acquire   func(OperationHybridLabels) prometheus.Observer
	Execution func(OperationHybridLabels) prometheus.Observer
	Miss      func(OperationHybridLabels) prometheus.Counter
	Hit       func(OperationHybridLabels) prometheus.Counter
}{
	Acquire:   func(l OperationHybridLabels) prometheus.Observer { return opHybridAcquire.WithLabelValues(l.values()...) },
	Execution: func(l OperationHybridLabels) prometheus.Observer { return opHybridExecution.WithLabelValues(l.values()...) },
	Miss:      func(l OperationHybridLabels) prometheus.Counter { return opHybridMiss.WithLabelValues(l.values()...) },
	Hit:       func(l OperationHybridLabels) prometheus.Counter { return opHybridHit.WithLabelValues(l.values()...) },
}

func init() {
	prometheus.MustRegister(poolHybridAccepted, poolHybridCurrent, opHybridAcquire, opHybridExecution, opHybridMiss, opHybridHit)
}
