package middleware

import "pggat2/lib/zap"

type Context interface {
	// Cancel the current packet
	Cancel()

	// Write packet to underlying connection
	Write(packet zap.Packet) error
}
