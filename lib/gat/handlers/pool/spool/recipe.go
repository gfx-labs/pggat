package spool

import "gfx.cafe/gfx/pggat/lib/gat/handlers/pool"

type Recipe struct {
	Name   string
	Recipe *pool.Recipe

	Penalty int

	Servers []*Server
}

func NewRecipe(name string, recipe *pool.Recipe) *Recipe {
	return &Recipe{
		Name:   name,
		Recipe: recipe,
	}
}

func (T *Recipe) Score() int {
	return T.Recipe.Priority + T.Penalty
}
