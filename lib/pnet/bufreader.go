package pnet

import (
	"pggat2/lib/pnet/packet"
	"pggat2/lib/util/decorator"
	"pggat2/lib/util/ring"
)

type bufIn struct {
	typ    packet.Type
	start  int
	length int
}

type BufReader struct {
	noCopy   decorator.NoCopy
	buf      packet.InBuf
	payloads []byte
	ins      ring.Ring[bufIn]
	reader   Reader
}

func MakeBufReader(reader Reader) BufReader {
	return BufReader{
		reader: reader,
	}
}

func NewBufReader(reader Reader) *BufReader {
	v := MakeBufReader(reader)
	return &v
}

func (T *BufReader) Buffer(in *packet.In) {
	if T.ins.Length() == 0 {
		// reset header
		T.payloads = T.payloads[:0]
	}
	start := len(T.payloads)
	full := in.Full()
	length := len(full)
	T.payloads = append(T.payloads, full...)
	T.ins.PushBack(bufIn{
		typ:    in.Type(),
		start:  start,
		length: length,
	})
}

func (T *BufReader) Read() (packet.In, error) {
	if in, ok := T.ins.PopFront(); ok {
		if in.typ == packet.None {
			panic("expected typed packet, got untyped")
		}
		T.buf.Reset(
			in.typ,
			T.payloads[in.start:in.start+in.length],
		)
		// returned buffered packet
		return packet.MakeIn(
			&T.buf,
		), nil
	}
	// fall back to underlying
	return T.reader.Read()
}

func (T *BufReader) ReadUntyped() (packet.In, error) {
	if in, ok := T.ins.PopFront(); ok {
		if in.typ != packet.None {
			panic("expected untyped packet, got typed")
		}
		T.buf.Reset(
			packet.None,
			T.payloads[in.start:in.start+in.length],
		)
		// returned buffered packet
		return packet.MakeIn(
			&T.buf,
		), nil
	}
	// fall back to underlying
	return T.reader.ReadUntyped()
}

var _ Reader = (*BufReader)(nil)
