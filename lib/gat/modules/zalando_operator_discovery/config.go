package zalando_operator_discovery

import (
	"gfx.cafe/util/go/gun"
	"k8s.io/client-go/rest"
)

type Config struct {
	Namespace                   string `env:"PGGAT_NAMESPACE" default:"default"`
	ConfigMapName               string `env:"CONFIG_MAP_NAME"`
	OperatorConfigurationObject string `env:"POSTGRES_OPERATOR_CONFIGURATION_OBJECT"`

	Rest *rest.Config
}

func Load() (Config, error) {
	var config Config
	gun.Load(&config)

	var err error
	config.Rest, err = rest.InClusterConfig()
	if err != nil {
		return Config{}, err
	}
	return config, nil
}
