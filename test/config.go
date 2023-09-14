package test

import (
	"pggat/lib/gat/pool/dialer"
)

type Config struct {
	Modes map[string]dialer.Dialer
}
