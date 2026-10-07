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
	// ServerError is the first ErrorResponse the server sent during query, if any.
	ServerError error
	TxState     byte

	// pending counts extended protocol requests (Parse, Bind, Close, Describe, Execute) whose terminal
	// response has not been read from the server. Reset at ReadyForQuery.
	pending int
	// failed is set once the server answered a request with ErrorResponse. The server then discards
	// every message, including Flush, until Sync, so no further responses are expected.
	failed bool
}

func (T *serverToPeerBinding) request() {
	if !T.failed {
		T.pending++
	}
}

// complete records the terminal response of the oldest pending request.
func (T *serverToPeerBinding) complete() {
	if T.pending > 0 {
		T.pending--
	}
}

// serverFailed records an ErrorResponse inside an extended protocol pipeline.
func (T *serverToPeerBinding) serverFailed() {
	T.pending = 0
	T.failed = true
}

// synced records ReadyForQuery.
func (T *serverToPeerBinding) synced() {
	T.pending = 0
	T.failed = false
}

func (T *serverToPeerBinding) ErrUnexpectedPacket() error {
	return ErrUnexpectedPacket(T.Packet.Type())
}

func (T *serverToPeerBinding) ServerRead(ctx context.Context) error {
	// Do not leave packets already received for the peer in its write buffer while waiting on a slow
	// server. While the next server packet is already buffered, keep batching so large results are not
	// written one packet at a time.
	if !T.Server.HasBufferedPacket() {
		T.PeerFlush(ctx)
	}

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

// PeerFlush sends buffered packets to the peer so early responses are not held back.
func (T *serverToPeerBinding) PeerFlush(ctx context.Context) {
	if T == nil || !T.PeerOK() {
		return
	}
	if err := T.Peer.Flush(ctx); err != nil {
		T.PeerFail(err)
	}
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
