package gat

type Listener interface {
	Module

	Endpoints() []Endpoint
}
