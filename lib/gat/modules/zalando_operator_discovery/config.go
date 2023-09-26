package zalando_operator_discovery

import (
	"gfx.cafe/util/go/gun"
	"k8s.io/client-go/rest"
)

type Config struct {
	Namespace                   string `env:"PGGAT_NAMESPACE" default:"default" json:"namespace"`
	ConfigMapName               string `env:"CONFIG_MAP_NAME" json:"config_map_name"`
	OperatorConfigurationObject string `env:"POSTGRES_OPERATOR_CONFIGURATION_OBJECT" json:"operator_configuration_object"`

	Rest *rest.Config `json:"-"`
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
