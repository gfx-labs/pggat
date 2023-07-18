package middleware

import "pggat2/lib/zap"

type Nil struct{}

func (Nil) Write(_ Context, _ zap.Inspector) error {
	return nil
}

func (Nil) Read(_ Context, _ zap.Inspector) error {
	return nil
}

var _ Middleware = Nil{}
