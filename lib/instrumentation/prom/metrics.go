package prom

import (
	"gfx.cafe/open/gotoprom"
	"github.com/prometheus/client_golang/prometheus"
)

type ListenerLabels struct {
	ListenAddr string `label:"listen_addr"`
}

var Listener struct {
	Incoming func(ListenerLabels) prometheus.Counter `name:"incoming"`
	Accepted func(ListenerLabels) prometheus.Counter `name:"accepted"`
	Client   func(ListenerLabels) prometheus.Gauge   `name:"client"`
}

type ServingLabels struct {
}

var Serving struct {
	Route func() `name:""`
}

type InstanceLabels struct {
}

var Instance struct {
	Route func() `name:""`
}

func init() {
	gotoprom.MustInit(&Listener, "pggat_listener", prometheus.Labels{})
	gotoprom.MustInit(&Instance, "pggat_instance", prometheus.Labels{})
	gotoprom.MustInit(&Serving, "pggat_serving", prometheus.Labels{})
}
