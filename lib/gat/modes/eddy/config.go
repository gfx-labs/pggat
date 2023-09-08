package eddy

import (
	"k8s.io/client-go/rest"
)

type Config struct {
	Namespace string

	Rest *rest.Config
}

func Load() (*Config, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return &Config{
		Namespace: "default",

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
