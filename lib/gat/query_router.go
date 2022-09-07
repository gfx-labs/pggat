package gat

import "gfx.cafe/gfx/pggat/lib/config"

type QueryRouter interface {
	InferRole(query string) (config.ServerRole, error)
}
