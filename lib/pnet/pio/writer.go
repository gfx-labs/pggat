package pio

import (
	"encoding/binary"
	"io"

	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
	"pggat2/lib/util/decorator"
)

type Writer struct {
	noCopy decorator.NoCopy
	writer io.Writer
	// header buffer for writing packet headers
	// (allocating within Write would escape to heap)
	header [5]byte

	buf packet.OutBuf
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

// Write gives you a packet.Out
// Calling Write will invalidate all other packet.Out's for this Writer
func (T *Writer) Write() packet.Out {
	T.buf.Reset()

	return packet.MakeOut(
		&T.buf,
	)
}

func (T *Writer) Send(typ packet.Type, payload []byte) error {
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

func (T *Writer) WriteByte(b byte) error {
	T.header[0] = b
	_, err := T.writer.Write(T.header[:1])
	return err
}

var _ pnet.Writer = (*Writer)(nil)
