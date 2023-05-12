package mw2

import "pggat2/lib/zap"

type Nil struct{}

func (Nil) Send(_ Context, _ zap.Out) error {
	return nil
}

func (Nil) Read(_ Context, _ zap.In) error {
	return nil
}

var _ Middleware = Nil{}
