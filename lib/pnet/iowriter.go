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
	header [5]byte

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
	T.buf.Reset()

	return packet.MakeOut(
		&T.buf,
		T,
	)
}

func (T *IOWriter) Send(typ packet.Type, payload []byte) error {
	/* if typ != packet.None {
		log.Printf("write typed packet %c %v\n", typ, payload)
	} else {
		log.Println("write untyped packet", payload)
	} */

	// prepare header
	T.header[0] = byte(typ)
	binary.BigEndian.PutUint32(T.header[1:], uint32(len(payload)+4))

	// write header
	if typ != packet.None {
		_, err := T.writer.Write(T.header[:])
		if err != nil {
			return err
		}
	} else {
		_, err := T.writer.Write(T.header[1:])
		if err != nil {
			return err
		}
	}

	// write payload
	_, err := T.writer.Write(payload)
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
