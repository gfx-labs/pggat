package test

import (
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/dialer"
)

type Config struct {
	Modes map[string]pool.Options
	Peer  dialer.Dialer
}
