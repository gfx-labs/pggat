package gat

type Stopper interface {
	Module

	Stop() error
}
