package pnet

import (
	"pggat2/lib/pnet/packet"
)

func PreRead(reader Reader) (Reader, error) {
	in, err := reader.Read()
	if err != nil {
		return nil, err
	}
	return newPolled(in, reader), nil
}

func PreReadUntyped(reader Reader) (Reader, error) {
	in, err := reader.ReadUntyped()
	if err != nil {
		return nil, err
	}
	return newPolled(in, reader), nil
}

type polled struct {
	in     packet.In
	read   bool
	reader Reader
}

func newPolled(in packet.In, reader Reader) *polled {
	return &polled{
		in:     in,
		reader: reader,
	}
}

func (T *polled) Read() (packet.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.reader.Read()
}

func (T *polled) ReadUntyped() (packet.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.reader.ReadUntyped()
}

var _ Reader = (*polled)(nil)
