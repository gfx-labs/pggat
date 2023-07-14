package middleware

import "pggat2/lib/zap"

type Context interface {
	// Cancel the current packet
	Cancel()

	// Send to underlying writer
	Send(out zap.Out) error
}
