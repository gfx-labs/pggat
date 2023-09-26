package gat

type Exposed interface {
	Module

	Endpoints() []Endpoint
}
