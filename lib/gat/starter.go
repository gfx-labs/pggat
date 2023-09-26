package gat

type Starter interface {
	Module

	Start() error
}
