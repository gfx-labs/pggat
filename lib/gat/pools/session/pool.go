package session

import (
	"log"
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
	for {
		clientErr, serverErr := bouncers.Bounce(client, server)
		if clientErr != nil || serverErr != nil {
			_ = client.Close()
			if serverErr == nil {
				T.release(server)
			} else {
				_ = server.Close()
			}
			break
		}
	}
}

func (T *Pool) AddRecipe(name string, recipe gat.Recipe) {
	for i := 0; i < recipe.GetMinConnections(); i++ {
		rw, err := recipe.Connect()
		if err != nil {
			// TODO(garet) do something here
			log.Printf("Failed to connect: %v", err)
			continue
		}
		err2 := backends.Accept(rw, recipe.GetUser(), recipe.GetPassword(), recipe.GetDatabase())
		if err2 != nil {
			_ = rw.Close()
			// TODO(garet) do something here
			log.Printf("Failed to connect: %v", err2)
			continue
		}
		T.release(rw)
	}
}

func (T *Pool) RemoveRecipe(name string) {
	panic("TODO")
	// TODO(garet)
}

var _ gat.Pool = (*Pool)(nil)
