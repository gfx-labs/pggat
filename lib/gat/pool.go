package gat

import "gfx.cafe/gfx/pggat/lib/config"

type Pool interface {
	GetUser(name string) (*config.User, error)
	GetRouter() QueryRouter
	WithUser(name string) (ConnectionPool, error)
}
