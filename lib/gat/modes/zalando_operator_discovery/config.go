package zalando_operator_discovery

import (
	"os"

	"k8s.io/client-go/rest"
)

type Config struct {
	Namespace                   string
	ConfigMapName               string
	OperatorConfigurationObject string

	Rest *rest.Config
}

func Load() (*Config, error) {
	namespace := os.Getenv("PGGAT_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	configMapName := os.Getenv("CONFIG_MAP_NAME")
	operatorConfigurationObject := os.Getenv("POSTGRES_OPERATOR_CONFIGURATION_OBJECT")

	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return &Config{
		Namespace:                   namespace,
		ConfigMapName:               configMapName,
		OperatorConfigurationObject: operatorConfigurationObject,

		Rest: restConfig,
	}, nil
}

func (T *Config) ListenAndServe() error {
	server, err := NewServer(T)
	if err != nil {
		return err
	}
	return server.ListenAndServe()
}
