package kitchen

import (
	"math"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
)

type Recipe struct {
	recipe  *pool.Recipe
	ratings []Rating
	score   int
	conns   map[*fed.Conn]struct{}
}

func NewRecipe(recipe *pool.Recipe, initial []*fed.Conn) *Recipe {
	conns := make(map[*fed.Conn]struct{}, len(initial))
	for _, conn := range initial {
		conns[conn] = struct{}{}
	}

	return &Recipe{
		recipe: recipe,
		conns:  conns,
	}
}

func (T *Recipe) Rating() int {
	if T.score == math.MaxInt {
		return T.score
	}
	return T.score + T.recipe.Priority
}
