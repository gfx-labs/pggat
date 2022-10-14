package metrics

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type gatMetrics struct {
	ConnectionCounter      prometheus.Counter
	ConnectionErrorCounter *prometheus.CounterVec
	ActiveConnections      prometheus.Gauge
}

func GatMetrics() *gatMetrics {
	s.Lock()
	defer s.Unlock()
	if s.gat == nil {
		s.gat = newGatmetrics()
	}
	return s.gat
}

func newGatmetrics() *gatMetrics {
	hostname := os.Getenv("HOSTNAME")
	if hostname == "" {
		hostname = "default"
	}
	o := &gatMetrics{
		ConnectionCounter: promauto.NewCounter(prometheus.CounterOpts{
			Name: "pggat_connection_count_total",
			Help: "total number of connections initiated with pggat",
			ConstLabels: prometheus.Labels{
				"pod": hostname,
			},
		}),
		ActiveConnections: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "pggat_current_connection_count",
			Help: "number of connections to pggat currently",
			ConstLabels: prometheus.Labels{
				"pod": hostname,
			},
		}),
	}
	return o
}

func RecordAcceptConnectionStatus(err error) {
	if !On() {
		return
	}
	g := GatMetrics()
	if err != nil {
		g.ConnectionErrorCounter.WithLabelValues(err.Error()).Inc()
	}
	g.ConnectionCounter.Inc()
}

func RecordActiveConnections(change int) {
	if !On() {
		return
	}
	g := GatMetrics()
	g.ActiveConnections.Add(float64(change))
}
