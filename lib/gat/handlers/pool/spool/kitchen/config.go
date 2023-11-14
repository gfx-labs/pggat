package kitchen

import (
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
)

type Config struct {
	Scorers []pool.Scorer
	Logger  *zap.Logger
}
