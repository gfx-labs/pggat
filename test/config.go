package test

import "gfx.cafe/gfx/pggat/lib/gat/handlers/pool"

type Config struct {
	// Stress is how many connections to run simultaneously for stress testing. <= 1 disables stress testing.
	Stress int

	Modes map[string]pool.Dialer
}
