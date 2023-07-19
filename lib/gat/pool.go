package gat

import "pggat2/lib/zap"

type Pool interface {
	Serve(client zap.ReadWriter)

	AddRecipe(name string, recipe Recipe)
	RemoveRecipe(name string)
}
