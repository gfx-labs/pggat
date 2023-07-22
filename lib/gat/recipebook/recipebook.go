package recipebook

import (
	"pggat2/lib/gat"
	"sync"
)

type Entry struct {
	r       gat.Recipe
	onEvict func()
}

type Book struct {
	item map[string]Entry
	mu   sync.RWMutex
}

func NewBook() *Book {
	return &Book{
		item: map[string]Entry{},
	}
}

func (b *Book) Remove(name string) (found bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	val, ok := b.item[name]
	if !ok {
		return false
	}
	val.onEvict()
	delete(b.item, name)
	return true
}

func (b *Book) AddIfNew(name string, recipe gat.Recipe, onEvict func()) (changed bool) {
	e := Entry{r: recipe, onEvict: onEvict}
	b.mu.Lock()
	defer b.mu.Unlock()
	val, ok := b.item[name]
	if !ok {
		b.item[name] = e
		return true
	}
	if !gat.RecipesEqual(val.r, recipe) {
		b.item[name] = e
		return true
	}
	return false
}
