package zalando_operator_discovery

import (
	"crypto/tls"
	"time"

	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/gat/modules/discovery"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func NewModule(config Config) (*discovery.Module, error) {
	d, err := NewDiscoverer(config)
	if err != nil {
		return nil, err
	}
	m, err := discovery.NewModule(discovery.Config{
		Discoverer:    d,
		ServerSSLMode: bouncer.SSLModePrefer,
		ServerSSLConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		ServerReconnectInitialTime: 5 * time.Second,
		ServerReconnectMaxTime:     5 * time.Second,
		ServerIdleTimeout:          5 * time.Minute,
		// ServerResetQuery: "discard all",
		TrackedParameters: []strutil.CIString{
			strutil.MakeCIString("client_encoding"),
			strutil.MakeCIString("datestyle"),
			strutil.MakeCIString("timezone"),
			strutil.MakeCIString("standard_conforming_strings"),
			strutil.MakeCIString("application_name"),
		},
		PoolMode: "transaction", // TODO(garet) pool mode from operator config
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}
