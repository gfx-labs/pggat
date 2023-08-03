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

func (T *Context) PeerRead(packet *zap.Packet) bool {
	if T == nil {
		return false
	}
	if !T.PeerOK() {
		return false
	}
	err := T.Peer.Read(packet)
	if err != nil {
		T.PeerFail(err)
		return false
	}
	return true
}

func (T *Context) PeerWrite(packet *zap.Packet) {
	if T == nil {
		return
	}
	if !T.PeerOK() {
		return
	}
	err := T.Peer.Write(packet)
	if err != nil {
		T.PeerFail(err)
	}
}

func (T *Context) PeerWriteV(packets *zap.Packets) {
	if T == nil {
		return
	}
	if !T.PeerOK() {
		return
	}
	err := T.Peer.WriteV(packets)
	if err != nil {
		T.PeerFail(err)
	}
}
