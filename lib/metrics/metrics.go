package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

func On() bool {
	return true
}

var s = &metrics{
	counters: map[string]prometheus.Counter{},
	histos:   map[string]prometheus.Histogram{},
	pools:    make(map[string]poolMetrics),
}

type metrics struct {
	counters map[string]prometheus.Counter
	histos   map[string]prometheus.Histogram
	pools    map[string]poolMetrics
	gat      *gatMetrics
	sync.RWMutex
}

func Counter(bucket string, name string, help string) prometheus.Counter {
	full := bucket + "_" + name
	s.Lock()
	c, ok := s.counters[full]
	if !ok {
		c = newCounter(bucket, name, help)
		s.counters[full] = c
	}
	s.Unlock()
	return c
}

func Hist(bucket string, name string, help string) prometheus.Histogram {
	full := bucket + "_" + name
	s.Lock()
	c, ok := s.histos[full]
	if !ok {
		c = newHistogram(bucket, name, help)
		s.histos[full] = c
	}
	s.Unlock()
	return c
}

func Inc(bucket string, name string, help string) {
	if !On() {
		return
	}
	Counter(bucket, name, help).Inc()
}
func Add(bucket string, name string, help string, entry float64) {
	if !On() {
		return
	}
	Hist(bucket, name, help).Observe(entry)
}

func init() {
	t1s := time.NewTicker(1 * time.Second)
	t15s := time.NewTicker(15 * time.Second)
	t1m := time.NewTicker(1 * time.Minute)
	t15m := time.NewTicker(15 * time.Minute)
	go func() {
		for {
			select {
			case <-t1s.C:
			case <-t15s.C:
			case <-t1m.C:
			case <-t15m.C:
			}
		}
	}()
}

func newCounter(app string, name string, help string) prometheus.Counter {
	return promauto.NewCounter(prometheus.CounterOpts{
		Name: app + "_" + name,
		Help: help,
	})
}

var defaultHist = []float64{0.001, 0.005, 0.01, 0.025, 0.035, 0.045, 0.05, 0.1, 0.15, 0.20, 0.25, 0.30, 0.35, 0.40, 0.45, 0.5, 1, 2.5, 5, 10, 20, 35, 50, 75, 100, 125, 250, 500, 750, 1000, 10000}

func newHistogram(app string, name string, help string) prometheus.Histogram {
	return promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    app + "_" + name,
		Help:    help,
		Buckets: defaultHist,
	})
}
