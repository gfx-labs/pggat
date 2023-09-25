package cloud_sql_discovery

import (
	"errors"

	"gfx.cafe/util/go/gun"
)

type Config struct {
	Project       string `env:"PGGAT_GC_PROJECT"`
	IpAddressType string `env:"PGGAT_GC_IP_ADDR_TYPE" default:"PRIMARY"`
	AuthUser      string `env:"PGGAT_GC_AUTH_USER" default:"pggat"`
	AuthPassword  string `env:"PGGAT_GC_AUTH_PASSWORD"`
}

func Load() (Config, error) {
	var conf Config
	gun.Load(&conf)
	if conf.Project == "" {
		return Config{}, errors.New("expected PGGAT_GC_PROJECT")
	}
	return conf, nil
}
