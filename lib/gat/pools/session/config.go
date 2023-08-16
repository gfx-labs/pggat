package session

import "pggat2/lib/gat"

type Config struct {
	gat.BaseRawPoolConfig

	// RoundRobin determines which order connections will be chosen. If false, connections are handled lifo,
	// otherwise they are chosen fifo
	RoundRobin bool
}
