package gat

type Pooler interface {
	NewPool() *Pool
}
