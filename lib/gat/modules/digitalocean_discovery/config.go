package digitalocean_discovery

import (
	"errors"

	"gfx.cafe/util/go/gun"
)

type Config struct {
	APIKey   string `env:"PGGAT_DO_API_KEY" json:"api_key"`
	Private  bool   `env:"PGGAT_DO_PRIVATE" json:"private"`
	PoolMode string `env:"PGGAT_POOL_MODE" json:"pool_mode"`
}

func Load() (Config, error) {
	var conf Config
	gun.Load(&conf)
	if conf.APIKey == "" {
		return Config{}, errors.New("expected auth token")
	}

	return conf, nil
}
