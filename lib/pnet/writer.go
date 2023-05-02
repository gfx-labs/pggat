package pnet

import (
	"encoding/binary"
	"io"

	"pggat2/lib/pnet/packet"
)

type Writer struct {
	writer io.Writer
	// buffer for writing packet headers
	// (allocating within Write would escape to heap)
	buffer [4]byte
}

func MakeWriter(writer io.Writer) Writer {
	return Writer{
		writer: writer,
	}
}

func NewWriter(writer io.Writer) *Writer {
	v := MakeWriter(writer)
	return &v
}

func (T *Writer) Write(raw packet.Raw) error {
	// write type byte
	err := T.WriteByte(byte(raw.Type))
	if err != nil {
		return err
	}

	return T.WriteUntyped(raw)
}

func (T *Writer) WriteUntyped(raw packet.Raw) error {
	// write len+4
	binary.BigEndian.PutUint32(T.buffer[:], uint32(len(raw.Payload)+4))
	_, err := T.writer.Write(T.buffer[:])
	if err != nil {
		return err
	}

	// write payload
	_, err = T.writer.Write(raw.Payload)
	if err != nil {
		return err
	}

	return nil
}

func (T *Writer) WriteByte(b byte) error {
	T.buffer[0] = b
	_, err := T.writer.Write(T.buffer[:1])
	return err
}
