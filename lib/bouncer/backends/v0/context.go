package backends

import (
	"context"
	"gfx.cafe/gfx/pggat/lib/fed"
)

type serverToPeerBinding struct {
	Server    *fed.Conn
	Peer      *fed.Conn
	Packet    fed.Packet
	PeerError error
	TxState   byte
}

func (T *serverToPeerBinding) ErrUnexpectedPacket() error {
	return ErrUnexpectedPacket(T.Packet.Type())
}

func (T *serverToPeerBinding) ServerRead(ctx context.Context) error {
	var err error
	T.Packet, err = T.Server.ReadPacket(ctx, true)
	return err
}

func (T *serverToPeerBinding) ServerWrite(ctx context.Context) error {
	return T.Server.WritePacket(ctx, T.Packet)
}

func (T *serverToPeerBinding) PeerOK() bool {
	if T == nil {
		return false
	}
	return T.Peer != nil && T.PeerError == nil
}

func (T *serverToPeerBinding) PeerFail(err error) {
	if T == nil {
		return
	}
	T.Peer = nil
	T.PeerError = err
}

func (T *serverToPeerBinding) PeerRead(ctx context.Context) bool {
	if T == nil {
		return false
	}
	if !T.PeerOK() {
		return false
	}
	var err error
	T.Packet, err = T.Peer.ReadPacket(ctx, true)
	if err != nil {
		T.PeerFail(err)
		return false
	}
	return true
}

func (T *serverToPeerBinding) PeerWrite(ctx context.Context) {
	if T == nil {
		return
	}
	if !T.PeerOK() {
		return
	}
	err := T.Peer.WritePacket(ctx, T.Packet)
	if err != nil {
		T.PeerFail(err)
	}
}
