package beforeexit

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
)

var (
	q      []func()
	active bool
	mu     sync.Mutex
)

func registerHandler() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c

		mu.Lock()
		defer mu.Unlock()
		for _, fn := range q {
			// ignore any panics in funcs
			func() {
				defer func() {
					recover()
				}()
				fn()
			}()
		}

		os.Exit(1)
	}()
}

// Run will register a func to run before exit on receiving an interrupt
// Tasks will run in the order that they are added
func Run(fn func()) {
	mu.Lock()
	defer mu.Unlock()

	q = append(q, fn)
	if !active {
		active = true
		registerHandler()
	}
}
