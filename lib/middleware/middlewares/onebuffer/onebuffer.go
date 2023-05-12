package onebuffer

import (
	"pggat2/lib/util/decorator"
	"pggat2/lib/zap"
)

type Onebuffer struct {
	noCopy decorator.NoCopy

	in   zap.In
	read bool
	zap.ReadWriter
}

func MakeOnebuffer(inner zap.ReadWriter) Onebuffer {
	return Onebuffer{
		read:       true,
		ReadWriter: inner,
	}
}

func (T *Onebuffer) Buffer() error {
	if !T.read {
		panic("a packet is already buffered in the Onebuffer!")
	}
	var err error
	T.in, err = T.ReadWriter.Read()
	T.read = false
	return err
}

func (T *Onebuffer) BufferUntyped() error {
	if !T.read {
		panic("a packet is already buffered in the Onebuffer!")
	}
	var err error
	T.in, err = T.ReadWriter.ReadUntyped()
	T.read = false
	return err
}

func (T *Onebuffer) Read() (zap.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.ReadWriter.Read()
}

func (T *Onebuffer) ReadUntyped() (zap.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.ReadWriter.ReadUntyped()
}
