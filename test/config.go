package test

import (
	"pggat/lib/gat/pool/dialer"
)

type Config struct {
	// Stress is how many connections to run simultaneously for stress testing. <= 1 disables stress testing.
	Stress int

	Modes map[string]dialer.Dialer
}
