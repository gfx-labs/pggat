package zap

import "io"

type ReadWriter interface {
	io.ByteReader
	Reader
	Writer
}
