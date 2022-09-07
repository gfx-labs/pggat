package gat

type Gat interface {
	GetPool(name string) (Pool, error)
}
