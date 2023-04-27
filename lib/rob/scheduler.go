package rob

type Scheduler interface {
	NewSink() Sink
	NewSource() Source
}
