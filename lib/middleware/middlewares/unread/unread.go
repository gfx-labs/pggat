package unread

import (
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
)

type Unread struct {
	in   packet.In
	read bool
	pnet.ReadWriter
}

func NewUnread(inner pnet.ReadWriter) (*Unread, error) {
	in, err := inner.Read()
	if err != nil {
		return nil, err
	}
	return &Unread{
		in:         in,
		ReadWriter: inner,
	}, nil
}

func NewUnreadUntyped(inner pnet.ReadWriter) (*Unread, error) {
	in, err := inner.ReadUntyped()
	if err != nil {
		return nil, err
	}
	return &Unread{
		in:         in,
		ReadWriter: inner,
	}, nil
}

func (T *Unread) Read() (packet.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.ReadWriter.Read()
}

func (T *Unread) ReadUntyped() (packet.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.ReadWriter.ReadUntyped()
}

var _ pnet.ReadWriter = (*Unread)(nil)
