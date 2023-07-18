package zap

type ReadWriter interface {
	Reader
	Writer
}

type CombinedReadWriter struct {
	Reader
	Writer
}
