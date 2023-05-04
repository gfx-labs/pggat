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
	buffer  [4]byte
	payload []byte
}

func MakeWriter(writer io.Writer) Writer {
	return Writer{
		writer:  writer,
		payload: make([]byte, 1024),
	}
}

func NewWriter(writer io.Writer) *Writer {
	v := MakeWriter(writer)
	return &v
}

func (T *Writer) Write() packet.Out {
	if T.payload == nil {
		panic("Previous Write was never finished")
	}

	payload := T.payload
	T.payload = nil
	return packet.MakeOut(
		payload[:0],
		T.write,
	)
}

func (T *Writer) write(typ packet.Type, payload []byte) error {
	T.payload = payload

	// write type byte (if present)
	if typ != packet.None {
		err := T.WriteByte(byte(typ))
		if err != nil {
			return err
		}
	}

	// write len+4
	binary.BigEndian.PutUint32(T.buffer[:], uint32(len(payload)+4))
	_, err := T.writer.Write(T.buffer[:])
	if err != nil {
		return err
	}

	// write payload
	_, err = T.writer.Write(payload)
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
