package rob

type Scheduler interface {
	// NewSink creates a new sink that fulfills input constraints
	NewSink(fulfills Constraints) Sink
	NewSource() Source
}
