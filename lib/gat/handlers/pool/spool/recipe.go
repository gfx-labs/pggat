package spool

import (
	"math"
	"time"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
)

type RecipeScore struct {
	Score      int
	Expiration time.Time
}

type Recipe struct {
	Name   string
	Recipe *pool.Recipe

	Scores []RecipeScore

	Servers []*Server
}

func NewRecipe(name string, recipe *pool.Recipe) *Recipe {
	return &Recipe{
		Name:   name,
		Recipe: recipe,
	}
}

func (T *Recipe) Priority() int {
	add := 0
	for _, score := range T.Scores {
		// return immediately so we don't overflow
		if score.Score == math.MaxInt {
			return math.MaxInt
		}
		add += score.Score
	}

	return T.Recipe.Priority + add
}
