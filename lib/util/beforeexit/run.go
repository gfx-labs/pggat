package beforeexit

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/google/uuid"

	"pggat2/lib/util/maps"
)

var toRun maps.RWLocked[uuid.UUID, func()]

// Run will register a func to run before exit on receiving an interrupt
// The order that tasks are run is undefined.
func Run(fn func()) uuid.UUID {
	id := uuid.New()
	toRun.Store(id, fn)
	return id
}

func Cancel(id uuid.UUID) {
	toRun.Delete(id)
}

func init() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c

		toRun.Range(func(_ uuid.UUID, fn func()) bool {
			// ignore any panics in funcs
			func() {
				defer func() {
					recover()
				}()
				fn()
			}()

			return true
		})

		os.Exit(1)
	}()
}
