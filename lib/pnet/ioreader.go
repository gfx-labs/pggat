package pnet

import (
	"encoding/binary"
	"io"

	"pggat2/lib/pnet/packet"
	"pggat2/lib/util/slices"
)

type IOReader struct {
	reader io.Reader
	// header buffer for reading packet headers
	// (allocating within Read would escape to heap)
	header [4]byte

	buf     packet.InBuf
	payload []byte
}

func MakeIOReader(reader io.Reader) IOReader {
	return IOReader{
		reader:  reader,
		payload: make([]byte, 1024),
	}
}

func NewIOReader(reader io.Reader) *IOReader {
	v := MakeIOReader(reader)
	return &v
}

// Read fetches the next packet from the underlying io.Reader and gives you a packet.In
// Calling Read will invalidate all other packet.In's for this IOReader
func (T *IOReader) Read() (packet.In, error) {
	typ, err := T.ReadByte()
	if err != nil {
		return packet.In{}, err
	}

	err = T.readPayload()
	if err != nil {
		return packet.In{}, err
	}

	T.buf.Reset(
		packet.Type(typ),
		T.payload,
	)

	return packet.MakeIn(
		&T.buf,
	), nil
}

// ReadUntyped is similar to Read, but it doesn't read a packet.Type
func (T *IOReader) ReadUntyped() (packet.In, error) {
	err := T.readPayload()
	if err != nil {
		return packet.In{}, err
	}

	T.buf.Reset(
		packet.None,
		T.payload,
	)

	return packet.MakeIn(
		&T.buf,
	), nil
}

func (T *IOReader) readPayload() error {
	if T.payload == nil {
		panic("Previous Read was never finished")
	}

	// read length int32
	_, err := io.ReadFull(T.reader, T.header[:])
	if err != nil {
		return err
	}

	length := binary.BigEndian.Uint32(T.header[:]) - 4

	// resize body to length
	T.payload = slices.Resize(T.payload, int(length))
	// read body
	_, err = io.ReadFull(T.reader, T.payload)
	if err != nil {
		return err
	}

	return nil
}

func (T *IOReader) ReadByte() (byte, error) {
	T.header[0] = 0
	_, err := io.ReadFull(T.reader, T.header[:1])
	return T.header[0], err
}
