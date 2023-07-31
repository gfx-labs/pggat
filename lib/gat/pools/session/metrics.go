package session

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type WorkerMetrics struct {
	LastActive time.Time
}

type Metrics struct {
	Workers map[uuid.UUID]WorkerMetrics
}

func (T *Metrics) InUse() int {
	var used int
	for _, worker := range T.Workers {
		if worker.LastActive == (time.Time{}) {
			used++
		}
	}
	return used
}

func (T *Metrics) String() string {
	return fmt.Sprintf("%d in use / %d total", T.InUse(), len(T.Workers))
}
