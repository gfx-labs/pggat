package prom

import "github.com/prometheus/client_golang/prometheus"

// Metric names, help text, labels and buckets match the series previously produced by gotoprom.

func newCounterVec(namespace, name, help string, labels ...string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(prometheus.CounterOpts{Namespace: namespace, Name: name, Help: help}, labels)
}

func newGaugeVec(namespace, name, help string, labels ...string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(prometheus.GaugeOpts{Namespace: namespace, Name: name, Help: help}, labels)
}

func newHistogramVec(namespace, name, help string, buckets []float64, labels ...string) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(prometheus.HistogramOpts{Namespace: namespace, Name: name, Help: help, Buckets: buckets}, labels)
}

var (
	acquireBuckets   = []float64{0.005, 0.01, 0.1, 0.25, 0.5, 0.75, 1, 5, 10, 100, 500, 1000, 5000}
	executionBuckets = []float64{1, 5, 10, 30, 75, 150, 300, 500, 1000, 2000, 5000, 7500, 10000, 15000, 30000}
)
