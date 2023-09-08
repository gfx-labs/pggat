package eddy

import (
	"k8s.io/client-go/rest"
)

type Config struct {
	rest *rest.Config
}

func Load() (*Config, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return &Config{
		rest: restConfig,
	}, nil
}

func (T *Config) ListenAndServe() error {
	server, err := NewServer(T)
	if err != nil {
		return err
	}
	return server.ListenAndServe()
}
