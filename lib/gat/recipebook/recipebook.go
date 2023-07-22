package recipebook

import (
	"pggat2/lib/gat"
	"sync"
)

type Book struct {
	item map[string]gat.Recipe
	mu   sync.RWMutex
}

func NewBook() *Book {
	return &Book{
		item: map[string]gat.Recipe{},
	}
}

func (b *Book) Remove(name string) (found bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	_, ok := b.item[name]
	if !ok {
		return false
	}
	delete(b.item, name)
	return true
}

func (b *Book) AddIfNew(name string, recipe gat.Recipe) (changed bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	val, ok := b.item[name]
	if !ok {
		b.item[name] = recipe
		return true
	}
	if !gat.RecipesEqual(val, recipe) {
		b.item[name] = recipe
		return true
	}
	return false
}
