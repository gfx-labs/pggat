package zalando_operator_discovery

import (
	"gfx.cafe/util/go/gun"
	"k8s.io/client-go/rest"
)

type Config struct {
	Namespace                   string `env:"PGGAT_NAMESPACE" default:"default"`
	ConfigMapName               string `env:"CONFIG_MAP_NAME"`
	OperatorConfigurationObject string `env:"POSTGRES_OPERATOR_CONFIGURATION_OBJECT"`
	TLSCrtFile                  string `env:"PGGAT_TLS_CRT_FILE" default:"/etc/ssl/certs/pgbouncer.crt"`
	TLSKeyFile                  string `env:"PGGAT_TLS_KEY_FILE" default:"/etc/ssl/certs/pgbouncer.key"`

	Rest *rest.Config
}

func Load() (*Config, error) {
	var config Config
	gun.Load(&config)

	var err error
	config.Rest, err = rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (T *Config) ListenAndServe() error {
	server, err := NewServer(T)
	if err != nil {
		return err
	}
	return server.ListenAndServe()
}
