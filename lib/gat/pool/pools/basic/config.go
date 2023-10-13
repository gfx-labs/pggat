package basic

import (
	"gfx.cafe/gfx/pggat/lib/gat/pool/scalingpool"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Config struct {
	// ReleaseAfterTransaction toggles whether servers should be released and re acquired after each transaction.
	// Use false for lower latency
	// Use true for better balancing
	ReleaseAfterTransaction bool

	// TrackedParameters are parameters which should be synced by updating the server, not the client.
	TrackedParameters []strutil.CIString

	scalingpool.Config
}
