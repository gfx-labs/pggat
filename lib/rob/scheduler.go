package rob

import "github.com/google/uuid"

type Scheduler interface {
	AddWorker(constraints Constraints, worker Worker) uuid.UUID
	GetWorker(id uuid.UUID) Worker
	RemoveWorker(id uuid.UUID) Worker
	WorkerCount() int

	NewSource() Worker

	ReadMetrics(metrics *Metrics)
}
