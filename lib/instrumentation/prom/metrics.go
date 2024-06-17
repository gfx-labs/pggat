package prom

import (
	"gfx.cafe/open/gotoprom"
	"github.com/prometheus/client_golang/prometheus"
)

type ListenerLabels struct {
	ListenAddr string `label:"listen_addr"`
}

var Listener struct {
	Incoming func(ListenerLabels) prometheus.Counter `name:"incoming" help:"incoming connections"`
	Accepted func(ListenerLabels) prometheus.Counter `name:"accepted" help:"accepted connetions"`
	Client   func(ListenerLabels) prometheus.Gauge   `name:"client" help:"current clients"`
}

type ServingLabels struct {
}

var Serving struct {
}

type InstanceLabels struct {
}

var Instance struct {
}

func init() {
	gotoprom.MustInit(&Listener, "pggat_listener", prometheus.Labels{})
	gotoprom.MustInit(&Instance, "pggat_instance", prometheus.Labels{})
	gotoprom.MustInit(&Serving, "pggat_serving", prometheus.Labels{})
}
