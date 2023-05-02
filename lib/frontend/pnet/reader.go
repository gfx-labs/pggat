package pnet

import (
	"encoding/binary"
	"io"

	"pggat2/lib/frontend/pnet/packet"
)

type Reader struct {
	reader io.Reader
	// buffer for reading packet headers
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
	// read type byte
	_, err := io.ReadFull(T.reader, T.buffer[:1])
	if err != nil {
		return packet.Raw{}, err
	}

	typ := packet.Type(T.buffer[0])

	pkt, err := T.ReadUntyped()
	if err != nil {
		return packet.Raw{}, err
	}

	pkt.Type = typ
	return pkt, nil
}

func (T *Reader) ReadUntyped() (packet.Raw, error) {
	pkt := packet.Raw{}

	// read length int32
	_, err := io.ReadFull(T.reader, T.buffer[:])
	if err != nil {
		return pkt, err
	}

	length := binary.BigEndian.Uint32(T.buffer[:]) - 4

	// read body
	pkt.Payload = make([]byte, length)
	_, err = io.ReadFull(T.reader, pkt.Payload)
	if err != nil {
		return pkt, err
	}

	return pkt, nil
}
