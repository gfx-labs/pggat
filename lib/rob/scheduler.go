package rob

type Scheduler interface {
	AddSink(Constraints, Worker)

	NewSource() Worker
}
