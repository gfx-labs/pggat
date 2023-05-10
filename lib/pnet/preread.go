package pnet

import (
	"pggat2/lib/pnet/packet"
)

// PreRead returns a buffered reader containing the first packet
// useful for waiting for a full packet before actually doing work
func PreRead(reader Reader) (Reader, error) {
	in, err := reader.Read()
	if err != nil {
		return nil, err
	}
	return newPolled(in, reader), nil
}

// PreReadUntyped does the same thing as PreReadUntyped but uses Reader.ReadUntyped
func PreReadUntyped(reader Reader) (Reader, error) {
	in, err := reader.ReadUntyped()
	if err != nil {
		return nil, err
	}
	return newPolled(in, reader), nil
}

type preRead struct {
	in     packet.In
	read   bool
	reader Reader
}

func newPolled(in packet.In, reader Reader) *preRead {
	return &preRead{
		in:     in,
		reader: reader,
	}
}

func (T *preRead) Read() (packet.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.reader.Read()
}

func (T *preRead) ReadUntyped() (packet.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.reader.ReadUntyped()
}

var _ Reader = (*preRead)(nil)
