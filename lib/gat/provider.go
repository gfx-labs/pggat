package gat

import "pggat/lib/gat/metrics"

// Provider provides pool to the server
type Provider interface {
	Module

	Lookup(user, database string) *Pool
	ReadMetrics(metrics *metrics.Pools)
}