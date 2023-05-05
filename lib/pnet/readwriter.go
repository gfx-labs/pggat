package pnet

type ReadWriter interface {
	Reader
	Writer
}

type JoinedReadWriter struct {
	Reader
	Writer
}
