package session

type Config struct {
	// RoundRobin determines which order connections will be chosen. If false, connections are handled lifo,
	// otherwise they are chosen fifo
	RoundRobin bool
}
