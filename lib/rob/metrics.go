package rob

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type WorkerMetrics struct {
	Idle   time.Duration
	Active time.Duration
}

type JobMetrics struct {
	Created    time.Time
	Backlogged bool
}

type Metrics struct {
	Jobs    map[uuid.UUID]JobMetrics
	Workers map[uuid.UUID]WorkerMetrics
}

func (T *Metrics) BackloggedJobCount() int {
	count := 0

	for _, job := range T.Jobs {
		if job.Backlogged {
			count++
		}
	}

	return count
}

func (T *Metrics) AverageJobAge() time.Duration {
	now := time.Now()

	sum := time.Duration(0)
	count := len(T.Jobs)

	for _, job := range T.Jobs {
		sum += now.Sub(job.Created)
	}

	if count == 0 {
		return 0
	}

	return sum / time.Duration(count)
}

func (T *Metrics) MaxJobAge() time.Duration {
	now := time.Now()

	max := time.Duration(0)

	for _, job := range T.Jobs {
		age := now.Sub(job.Created)
		if age > max {
			max = age
		}
	}

	return max
}

func (T *Metrics) AverageWorkerUtilization() float64 {
	idle := time.Duration(0)
	active := time.Duration(0)

	for _, worker := range T.Workers {
		idle += worker.Idle
		active += worker.Active
	}

	return float64(active) / float64(idle+active)
}

func (T *Metrics) String() string {
	return fmt.Sprintf("%d queued jobs (%d backlogged, %s avg age, %s max age) / %d workers (%.2f%% util)", len(T.Jobs), T.BackloggedJobCount(), T.AverageJobAge().String(), T.MaxJobAge().String(), len(T.Workers), T.AverageWorkerUtilization()*100)
}
