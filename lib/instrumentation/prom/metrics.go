package prom

import (
	"github.com/prometheus/client_golang/prometheus"
)

type ListenerLabels struct {
	ListenAddr string
}

var (
	listenerIncoming = newCounterVec("pggat_listener", "incoming", "incoming connections", "listen_addr")
	listenerAccepted = newCounterVec("pggat_listener", "accepted", "accepted connetions", "listen_addr")
	listenerClient   = newGaugeVec("pggat_listener", "client", "current clients", "listen_addr")
)

var Listener = struct {
	Incoming func(ListenerLabels) prometheus.Counter
	Accepted func(ListenerLabels) prometheus.Counter
	Client   func(ListenerLabels) prometheus.Gauge
}{
	Incoming: func(l ListenerLabels) prometheus.Counter { return listenerIncoming.WithLabelValues(l.ListenAddr) },
	Accepted: func(l ListenerLabels) prometheus.Counter { return listenerAccepted.WithLabelValues(l.ListenAddr) },
	Client:   func(l ListenerLabels) prometheus.Gauge { return listenerClient.WithLabelValues(l.ListenAddr) },
}

func init() {
	prometheus.MustRegister(listenerIncoming, listenerAccepted, listenerClient)
}
