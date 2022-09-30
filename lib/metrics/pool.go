package metrics

import (
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
			Buckets: taskDurationBucketsUs,
			ConstLabels: prometheus.Labels{
				"db":   db,
				"user": user,
			},
		}, []string{}),
		QueryLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "pggat_db_query_latency",
			Help:    "query latency",
			Buckets: taskDurationBucketsUs,
			ConstLabels: prometheus.Labels{
				"db":   db,
				"user": user,
			},
		}, []string{}),
		WaitLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "pggat_db_wait_latency",
			Help:    "wait latency",
			Buckets: taskDurationBucketsUs,
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
			Name: "pggat_received_bytes_total",
			Help: "total number of bytes received",
			ConstLabels: prometheus.Labels{
				"db":   db,
				"user": user,
			},
		}, []string{}),
	}
	return o
}
