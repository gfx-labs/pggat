package rob

import "github.com/google/uuid"

type Scheduler interface {
	AddSink(constraints Constraints, worker Worker) uuid.UUID
	GetSink(id uuid.UUID) Worker
	RemoveSink(id uuid.UUID) Worker

	NewSource() Worker

	ReadMetrics(metrics *Metrics)
}
