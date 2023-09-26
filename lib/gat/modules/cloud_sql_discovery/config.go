package cloud_sql_discovery

import (
	"errors"

	"gfx.cafe/util/go/gun"
)

type Config struct {
	Project       string `env:"PGGAT_GC_PROJECT" json:"project"`
	IpAddressType string `env:"PGGAT_GC_IP_ADDR_TYPE" default:"PRIMARY" json:"ip_address_type"`
	AuthUser      string `env:"PGGAT_GC_AUTH_USER" default:"pggat" json:"auth_user"`
	AuthPassword  string `env:"PGGAT_GC_AUTH_PASSWORD" json:"auth_password"`
}

func Load() (Config, error) {
	var conf Config
	gun.Load(&conf)
	if conf.Project == "" {
		return Config{}, errors.New("expected PGGAT_GC_PROJECT")
	}
	return conf, nil
}
