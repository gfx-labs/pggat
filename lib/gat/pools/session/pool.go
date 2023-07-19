package session

import (
	"net"
	"sync"

	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/gat"
	"pggat2/lib/zap"
)

type Pool struct {
	// use slice lifo for better perf
	queue []zap.ReadWriter
	mu    sync.RWMutex

	signal chan struct{}
}

func NewPool() *Pool {
	return &Pool{
		signal: make(chan struct{}),
	}
}

func (T *Pool) acquire() zap.ReadWriter {
	for {
		T.mu.Lock()
		if len(T.queue) > 0 {
			server := T.queue[len(T.queue)-1]
			T.queue = T.queue[:len(T.queue)-1]
			T.mu.Unlock()
			return server
		}
		T.mu.Unlock()
		<-T.signal
	}
}

func (T *Pool) release(server zap.ReadWriter) {
	T.mu.Lock()
	defer T.mu.Unlock()
	T.queue = append(T.queue, server)

	select {
	case T.signal <- struct{}{}:
	default:
	}
}

func (T *Pool) Serve(client zap.ReadWriter) {
	server := T.acquire()
	defer T.release(server)
	for {
		if err := client.Poll(); err != nil {
			break
		}
		bouncers.Bounce(client, server)
	}
}

func (T *Pool) AddRecipe(name string, recipe gat.Recipe) {
	for i := 0; i < recipe.MinConnections; i++ {
		conn, err := net.Dial("tcp", recipe.Address)
		if err != nil {
			// TODO(garet) do something here
			continue
		}
		rw := zap.CombinedReadWriter{
			Reader: zap.IOReader{Reader: conn},
			Writer: zap.IOWriter{Writer: conn},
		}
		backends.Accept(rw, recipe.User, recipe.Password, recipe.Database)
		T.release(rw)
	}
}

func (T *Pool) RemoveRecipe(name string) {
	// TODO implement me
	panic("implement me")
}

var _ gat.Pool = (*Pool)(nil)
