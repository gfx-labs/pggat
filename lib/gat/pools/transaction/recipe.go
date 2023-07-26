package transaction

import (
	"sync"

	"github.com/google/uuid"

	"pggat2/lib/gat"
)

type Recipe struct {
	recipe gat.Recipe

	open []uuid.UUID
	mu   sync.Mutex
}

func NewRecipe(recipe gat.Recipe) *Recipe {
	return &Recipe{
		recipe: recipe,
	}
}
