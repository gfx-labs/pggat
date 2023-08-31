package middleware

import "pggat2/lib/fed"

type Middleware interface {
	Read(ctx Context, packet fed.Packet) error
	Write(ctx Context, packet fed.Packet) error
}
