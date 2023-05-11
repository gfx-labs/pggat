package zap

import "io"

type Reader interface {
	io.ByteReader

	Read() (In, error)
	ReadUntyped() (In, error)
}
