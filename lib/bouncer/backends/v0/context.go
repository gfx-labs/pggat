package backends

import "pggat2/lib/zap"

type Context struct {
	Peer      zap.ReadWriter
	PeerError error
	TxState   byte
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

func (T *Context) PeerRead() zap.Packet {
	if T == nil {
		return nil
	}
	if !T.PeerOK() {
		return nil
	}
	packet, err := T.Peer.ReadPacket(true)
	if err != nil {
		T.PeerFail(err)
		return nil
	}
	return packet
}

func (T *Context) PeerWrite(packet zap.Packet) {
	if T == nil {
		return
	}
	if !T.PeerOK() {
		return
	}
	err := T.Peer.WritePacket(packet)
	if err != nil {
		T.PeerFail(err)
	}
}
