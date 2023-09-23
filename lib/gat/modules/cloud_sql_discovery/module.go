package cloud_sql_discovery

import (
	"pggat/lib/gat/modules/discovery"
)

func NewModule(config Config) (*discovery.Module, error) {
	return discovery.NewModule(discovery.Config{})
}
