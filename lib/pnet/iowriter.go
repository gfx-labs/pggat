package pnet

import (
	"encoding/binary"
	"io"

	"pggat2/lib/pnet/packet"
	"pggat2/lib/util/decorator"
)

type IOWriter struct {
	noCopy decorator.NoCopy
	writer io.Writer
	// header buffer for writing packet headers
	// (allocating within Write would escape to heap)
	header [4]byte

	buf packet.OutBuf
}

func MakeIOWriter(writer io.Writer) IOWriter {
	return IOWriter{
		writer: writer,
	}
}

func NewIOWriter(writer io.Writer) *IOWriter {
	v := MakeIOWriter(writer)
	return &v
}

// Write gives you a packet.Out
// Calling Write will invalidate all other packet.Out's for this IOWriter
func (T *IOWriter) Write() packet.Out {
	if !T.buf.Initialized() {
		T.buf.Initialize(T.write)
	}
	T.buf.Reset()

	return packet.MakeOut(
		&T.buf,
	)
}

func (T *IOWriter) write(typ packet.Type, payload []byte) error {
	// write type byte (if present)
	if typ != packet.None {
		err := T.WriteByte(byte(typ))
		if err != nil {
			return err
		}
	}

	// write len+4
	binary.BigEndian.PutUint32(T.header[:], uint32(len(payload)+4))
	_, err := T.writer.Write(T.header[:])
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

func (T *IOWriter) WriteByte(b byte) error {
	T.header[0] = b
	_, err := T.writer.Write(T.header[:1])
	return err
}
