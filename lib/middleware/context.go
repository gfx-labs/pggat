package middleware

import "pggat/lib/fed"

type Context interface {
	// Cancel the current packet
	Cancel()

	// Write packet to underlying connection
	Write(packet fed.Packet) error
}
