package middleware

import "pggat/lib/fed"

type Nil struct{}

func (Nil) Read(_ Context, _ fed.Packet) error {
	return nil
}

func (Nil) Write(_ Context, _ fed.Packet) error {
	return nil
}

var _ Middleware = Nil{}
