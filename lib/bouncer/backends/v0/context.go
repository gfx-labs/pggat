package backends

import (
	"pggat/lib/fed"
)

type AcceptContext struct {
	Packet  fed.Packet
	Conn    fed.Conn
	Options AcceptOptions
}

type Context struct {
	Server    fed.ReadWriter
	Packet    fed.Packet
	Peer      fed.ReadWriter
	PeerError error
	TxState   byte
}

func (T *Context) ServerRead() error {
	var err error
	T.Packet, err = T.Server.ReadPacket(true, T.Packet)
	return err
}

func (T *Context) ServerWrite() error {
	return T.Server.WritePacket(T.Packet)
}

func (T *Context) PeerOK() bool {
	if T == nil {
		return false
	}
	return T.Peer != nil && T.PeerError == nil
}

func (T *Context) PeerFail(err error) {
	if T == nil {
		return
	}
	T.Peer = nil
	T.PeerError = err
}

func (T *Context) PeerRead() bool {
	if T == nil {
		return false
	}
	if !T.PeerOK() {
		return false
	}
	var err error
	T.Packet, err = T.Peer.ReadPacket(true, T.Packet)
	if err != nil {
		T.PeerFail(err)
		return false
	}
	return true
}

func (T *Context) PeerWrite() {
	if T == nil {
		return
	}
	if !T.PeerOK() {
		return
	}
	err := T.Peer.WritePacket(T.Packet)
	if err != nil {
		T.PeerFail(err)
	}
}
