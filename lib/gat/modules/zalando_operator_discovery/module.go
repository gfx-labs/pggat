package zalando_operator_discovery

import (
	"crypto/tls"
	"time"

	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/modules/discovery"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	gat.RegisterModule((*Module)(nil))
}

type Module struct {
	Config

	discovery.Module `json:"-"`
}

func (*Module) GatModule() gat.ModuleInfo {
	return gat.ModuleInfo{
		ID: "zalando_operator_discovery",
		New: func() gat.Module {
			return new(Module)
		},
	}
}

func (T *Module) Start() error {
	d, err := NewDiscoverer(T.Config)
	if err != nil {
		return err
	}

	T.Module = discovery.Module{
		Config: discovery.Config{
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
		},
	}

	return T.Module.Start()
}

var _ gat.Starter = (*Module)(nil)
