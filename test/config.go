package test

import (
	"gfx.cafe/gfx/pggat/lib/gat/pool/recipe"
)

type Config struct {
	// Stress is how many connections to run simultaneously for stress testing. <= 1 disables stress testing.
	Stress int

	Modes map[string]recipe.Dialer
}
