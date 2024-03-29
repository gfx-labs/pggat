package backends

import (
	"gfx.cafe/gfx/pggat/lib/fed"
)

type acceptContext struct {
	Conn    *fed.Conn
	Options acceptOptions
}

type context struct {
	Server    *fed.Conn
	Packet    fed.Packet
	Peer      *fed.Conn
	PeerError error
	TxState   byte
}

func (T *context) ErrUnexpectedPacket() error {
	return ErrUnexpectedPacket(T.Packet.Type())
}

func (T *context) ServerRead() error {
	var err error
	T.Packet, err = T.Server.ReadPacket(true)
	return err
}

func (T *context) ServerWrite() error {
	return T.Server.WritePacket(T.Packet)
}

func (T *context) PeerOK() bool {
	if T == nil {
		return false
	}
	return T.Peer != nil && T.PeerError == nil
}

func (T *context) PeerFail(err error) {
	if T == nil {
		return
	}
	T.Peer = nil
	T.PeerError = err
}

func (T *context) PeerRead() bool {
	if T == nil {
		return false
	}
	if !T.PeerOK() {
		return false
	}
	var err error
	T.Packet, err = T.Peer.ReadPacket(true)
	if err != nil {
		T.PeerFail(err)
		return false
	}
	return true
}

func (T *context) PeerWrite() {
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
