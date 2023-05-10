package unread

import (
	"pggat2/lib/pnet"
	"pggat2/lib/pnet/packet"
)

type Unread struct {
	in    packet.In
	read  bool
	inner pnet.ReadWriteSender
}

func NewUnread(inner pnet.ReadWriteSender) (*Unread, error) {
	in, err := inner.Read()
	if err != nil {
		return nil, err
	}
	return &Unread{
		in:    in,
		inner: inner,
	}, nil
}

func NewUnreadUntyped(inner pnet.ReadWriteSender) (*Unread, error) {
	in, err := inner.ReadUntyped()
	if err != nil {
		return nil, err
	}
	return &Unread{
		in:    in,
		inner: inner,
	}, nil
}

func (T *Unread) Read() (packet.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.inner.Read()
}

func (T *Unread) ReadUntyped() (packet.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.inner.ReadUntyped()
}

func (T *Unread) Write() packet.Out {
	return T.inner.Write()
}

func (T *Unread) Send(typ packet.Type, bytes []byte) error {
	return T.inner.Send(typ, bytes)
}

var _ pnet.ReadWriteSender = (*Unread)(nil)
