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
	buffer [4]byte
}

func MakeReader(reader io.Reader) Reader {
	return Reader{
		reader: reader,
	}
}

func NewReader(reader io.Reader) *Reader {
	v := MakeReader(reader)
	return &v
}

func (T *Reader) Read() (packet.Raw, error) {
	raw := packet.Raw{}
	err := T.ReadInto(&raw)
	return raw, err
}

func (T *Reader) ReadInto(raw *packet.Raw) error {
	// read type byte
	typ, err := T.ReadByte()
	if err != nil {
		return err
	}
	raw.Type = packet.Type(typ)

	err = T.ReadUntypedInto(raw)
	if err != nil {
		return err
	}

	return nil
}

func (T *Reader) ReadUntyped() (packet.Raw, error) {
	pkt := packet.Raw{}
	err := T.ReadUntypedInto(&pkt)
	return pkt, err
}

func (T *Reader) ReadUntypedInto(raw *packet.Raw) error {
	// read length int32
	_, err := io.ReadFull(T.reader, T.buffer[:])
	if err != nil {
		return err
	}

	length := binary.BigEndian.Uint32(T.buffer[:]) - 4

	// resize body to length
	raw.Payload = slices.Resize(raw.Payload, int(length))
	// read body
	_, err = io.ReadFull(T.reader, raw.Payload)
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
