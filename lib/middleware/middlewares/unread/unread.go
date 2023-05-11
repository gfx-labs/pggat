package unread

import (
	"pggat2/lib/zap"
)

type Unread struct {
	in   zap.In
	read bool
	zap.ReadWriter
}

func NewUnread(inner zap.ReadWriter) (*Unread, error) {
	in, err := inner.Read()
	if err != nil {
		return nil, err
	}
	return &Unread{
		in:         in,
		ReadWriter: inner,
	}, nil
}

func NewUnreadUntyped(inner zap.ReadWriter) (*Unread, error) {
	in, err := inner.ReadUntyped()
	if err != nil {
		return nil, err
	}
	return &Unread{
		in:         in,
		ReadWriter: inner,
	}, nil
}

func (T *Unread) Read() (zap.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.ReadWriter.Read()
}

func (T *Unread) ReadUntyped() (zap.In, error) {
	if !T.read {
		T.read = true
		return T.in, nil
	}
	return T.ReadWriter.ReadUntyped()
}

var _ zap.ReadWriter = (*Unread)(nil)
