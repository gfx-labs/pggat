package pnet

import (
	"encoding/binary"
	"io"

	"pggat2/lib/pnet/packet"
	"pggat2/lib/util/slices"
)

type Reader struct {
	reader io.Reader
	// buffer for reading packet headers
	// (allocating within Read would escape to heap)
	buffer  [4]byte
	payload []byte
}

func MakeReader(reader io.Reader) Reader {
	return Reader{
		reader:  reader,
		payload: make([]byte, 1024),
	}
}

func NewReader(reader io.Reader) *Reader {
	v := MakeReader(reader)
	return &v
}

func (T *Reader) Read() (packet.In, error) {
	typ, err := T.ReadByte()
	if err != nil {
		return packet.In{}, err
	}

	err = T.readPayload()
	if err != nil {
		return packet.In{}, err
	}

	payload := T.payload
	T.payload = nil

	return packet.MakeIn(
		packet.Type(typ),
		payload,
		func(payload []byte) {
			T.payload = payload
		},
	), nil
}

func (T *Reader) ReadUntyped() (packet.In, error) {
	err := T.readPayload()
	if err != nil {
		return packet.In{}, err
	}

	payload := T.payload
	T.payload = nil
	return packet.MakeIn(
		packet.None,
		payload,
		func(bytes []byte) {
			T.payload = payload
		},
	), nil
}

func (T *Reader) readPayload() error {
	if T.payload == nil {
		panic("Previous Read was never finished")
	}

	// read length int32
	_, err := io.ReadFull(T.reader, T.buffer[:])
	if err != nil {
		return err
	}

	length := binary.BigEndian.Uint32(T.buffer[:]) - 4

	// resize body to length
	T.payload = slices.Resize(T.payload, int(length))
	// read body
	_, err = io.ReadFull(T.reader, T.payload)
	if err != nil {
		return err
	}

	return nil
}

func (T *Reader) ReadByte() (byte, error) {
	T.buffer[0] = 0
	_, err := io.ReadFull(T.reader, T.buffer[:1])
	return T.buffer[0], err
}
