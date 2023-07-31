package rob

import (
	"time"

	"github.com/google/uuid"
)

type Scheduler interface {
	AddWorker(constraints Constraints, worker Worker) uuid.UUID
	GetWorker(id uuid.UUID) Worker
	GetIdleWorker() (uuid.UUID, time.Time)
	RemoveWorker(id uuid.UUID) Worker
	WorkerCount() int

	NewSource() Worker

	ReadMetrics(metrics *Metrics)
}
