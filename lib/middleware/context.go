package middleware

import "pggat2/lib/zap"

type Context interface {
	// Cancel the current packet
	Cancel()

	// Write packet to underlying connection
	Write(packet *zap.Packet) error
	// WriteUntyped is the same as Write but with an UntypedPacket
	WriteUntyped(packet *zap.UntypedPacket) error
}
