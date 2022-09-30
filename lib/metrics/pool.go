package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type poolMetrics struct {
	name          string
	TxLatency     *prometheus.HistogramVec
	QueryLatency  *prometheus.HistogramVec
	WaitLatency   *prometheus.HistogramVec
	ReceivedBytes *prometheus.CounterVec
	SentBytes     *prometheus.CounterVec
}

func PoolMetrics(db string, user string) poolMetrics {
	s.Lock()
	defer s.Unlock()
	pool, ok := s.pools[db+user]
	if !ok {
		pool = newPoolMetrics(db, user)
		s.pools[db+user] = pool
	}
	return pool
}

func newPoolMetrics(db string, user string) poolMetrics {
	o := poolMetrics{
		name: db + user,
		TxLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "pggat_db_transaction_latency",
			Help:    "transaction latency",
			Buckets: bucketsUs,
			ConstLabels: prometheus.Labels{
				"db":   db,
				"user": user,
			},
		}, []string{}),
		QueryLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "pggat_db_query_latency",
			Help:    "query latency",
			Buckets: bucketsUs,
			ConstLabels: prometheus.Labels{
				"db":   db,
				"user": user,
			},
		}, []string{}),
		WaitLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "pggat_db_wait_latency",
			Help:    "wait latency",
			Buckets: bucketsUs,
			ConstLabels: prometheus.Labels{
				"db":   db,
				"user": user,
			},
		}, []string{}),
		ReceivedBytes: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "pggat_received_bytes_total",
			Help: "total number of bytes received",
			ConstLabels: prometheus.Labels{
				"db":   db,
				"user": user,
			},
		}, []string{}),
		SentBytes: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "pggat_sent_bytes_total",
			Help: "total number of bytes received",
			ConstLabels: prometheus.Labels{
				"db":   db,
				"user": user,
			},
		}, []string{}),
	}
	return o
}

func RecordBytes(db string, user string, sent, received int64) {
	if !On() {
		return
	}
	p := PoolMetrics(db, user)
	p.SentBytes.WithLabelValues().Add(float64(sent))
	p.ReceivedBytes.WithLabelValues().Add(float64(received))
}

func RecordQueryTime(db string, user string, dur time.Duration) {
	if !On() {
		return
	}
	p := PoolMetrics(db, user)
	p.QueryLatency.WithLabelValues().Observe(float64(dur.Nanoseconds()))
}

func RecordTransactionTime(db string, user string, dur time.Duration) {
	if !On() {
		return
	}
	p := PoolMetrics(db, user)
	p.TxLatency.WithLabelValues().Observe(float64(dur.Nanoseconds()))
}

func RecordWaitTime(db string, user string, dur time.Duration) {
	if !On() {
		return
	}
	p := PoolMetrics(db, user)
	p.WaitLatency.WithLabelValues().Observe(float64(dur.Nanoseconds()))
}
