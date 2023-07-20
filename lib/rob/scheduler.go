package rob

import "github.com/google/uuid"

type Scheduler interface {
	AddSink(Constraints, Worker) uuid.UUID
	RemoveSink(id uuid.UUID)

	NewSource() Worker

	ReadMetrics() Metrics
}
