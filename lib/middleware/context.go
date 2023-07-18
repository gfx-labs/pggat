package middleware

import "pggat2/lib/zap"

type Context interface {
	// Cancel the current packet
	Cancel()

	BuildBefore(typed bool) zap.Builder
	BuildAfter(typed bool) zap.Builder
}
