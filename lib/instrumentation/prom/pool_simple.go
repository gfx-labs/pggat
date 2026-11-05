package prom

import (
	"github.com/prometheus/client_golang/prometheus"
)

type PoolSimpleLabels struct {
	Mode     string
	Database string
	User     string
}

func (s *PoolSimpleLabels) ToOperation() OperationSimpleLabels {
	return OperationSimpleLabels{
		Pool:     "basic",
		Mode:     s.Mode,
		Database: s.Database,
	}
}

type OperationSimpleLabels struct {
	Pool     string
	Mode     string
	Database string
	User     string
}

var (
	poolSimpleAccepted = newCounterVec("pggat_pool_simple", "accepted", "simple connections accepted", "mode", "database", "user")
	poolSimpleCurrent  = newGaugeVec("pggat_pool_simple", "current", "current simple connections", "mode", "database", "user")

	// The basic pool acquire histogram never had the 5000 bucket.
	opSimpleAcquire   = newHistogramVec("pggat_operation_simple", "acquire_ms", "ms to acquire from pool", acquireBuckets[:len(acquireBuckets)-1], "pool", "mode", "database", "user")
	opSimpleExecution = newHistogramVec("pggat_operation_simple", "execution_ms", "ms that the txn took to execute on remote", executionBuckets, "pool", "mode", "database", "user")
)

var PoolSimple = struct {
	Accepted func(PoolSimpleLabels) prometheus.Counter
	Current  func(PoolSimpleLabels) prometheus.Gauge
}{
	Accepted: func(l PoolSimpleLabels) prometheus.Counter {
		return poolSimpleAccepted.WithLabelValues(l.Mode, l.Database, l.User)
	},
	Current: func(l PoolSimpleLabels) prometheus.Gauge {
		return poolSimpleCurrent.WithLabelValues(l.Mode, l.Database, l.User)
	},
}

var OperationSimple = struct {
	Acquire   func(OperationSimpleLabels) prometheus.Observer
	Execution func(OperationSimpleLabels) prometheus.Observer
}{
	Acquire: func(l OperationSimpleLabels) prometheus.Observer {
		return opSimpleAcquire.WithLabelValues(l.Pool, l.Mode, l.Database, l.User)
	},
	Execution: func(l OperationSimpleLabels) prometheus.Observer {
		return opSimpleExecution.WithLabelValues(l.Pool, l.Mode, l.Database, l.User)
	},
}

func init() {
	prometheus.MustRegister(poolSimpleAccepted, poolSimpleCurrent, opSimpleAcquire, opSimpleExecution)
}
