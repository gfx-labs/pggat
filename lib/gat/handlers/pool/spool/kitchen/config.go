package kitchen

import (
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
)

type Config struct {
	Critics []pool.Critic
	Logger  *zap.Logger
}
