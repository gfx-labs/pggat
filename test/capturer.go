package test

import (
	"bytes"
	"fmt"

	"pggat/lib/fed"
	"pggat/lib/gsql"
)

type Capturer struct {
	Packets []fed.Packet
}

func (T *Capturer) WritePacket(packet fed.Packet) error {
	T.Packets = append(T.Packets, packet)
	return nil
}

func (T *Capturer) Check(other *Capturer) error {
	if len(T.Packets) != len(other.Packets) {
		return fmt.Errorf("not enough packets! got %d but expected %d", len(other.Packets), len(T.Packets))
	}

	for i := range T.Packets {
		expected := T.Packets[i]
		actual := other.Packets[i]

		if !bytes.Equal(expected.Bytes(), actual.Bytes()) {
			return fmt.Errorf("mismatched packet! expected %v but got %v", expected.Bytes(), actual.Bytes())
		}
	}

	return nil
}

var _ gsql.ResultWriter = (*Capturer)(nil)
