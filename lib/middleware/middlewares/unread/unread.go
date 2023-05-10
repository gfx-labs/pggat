package unread

import (
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
)

type Unread struct {
	in   packet.In
	read bool
	pnet.ReadWriteSender
}

func NewUnread(inner pnet.ReadWriteSender) (*Unread, error) {
	in, err := inner.Read()
	if err != nil {
		return nil, err
	}
	return &Unread{
		in:              in,
		ReadWriteSender: inner,
	}, nil
}

func NewUnreadUntyped(inner pnet.ReadWriteSender) (*Unread, error) {
	in, err := inner.ReadUntyped()
	if err != nil {
		return nil, err
	}
	return &Unread{
		in:              in,
		ReadWriteSender: inner,
	}, nil
}

func (T *Unread) Read() (packet.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.ReadWriteSender.Read()
}

func (T *Unread) ReadUntyped() (packet.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.ReadWriteSender.ReadUntyped()
}

var _ pnet.ReadWriteSender = (*Unread)(nil)
