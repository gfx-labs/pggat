package rob

type Metrics struct {
	JobsWaiting int
	JobsRunning int

	TotalWorkers  int
	WorkersActive int
	WorkersIdle   int
}
