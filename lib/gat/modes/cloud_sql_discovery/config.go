package cloud_sql_discovery

import (
	"errors"
	"time"

	"gfx.cafe/util/go/gun"
	"tuxpa.in/a/zlog/log"

	"pggat/lib/bouncer/frontends/v0"
	"pggat/lib/gat"
	"pggat/lib/gat/metrics"
	"pggat/lib/util/flip"
	"pggat/lib/util/strutil"
)

type Config struct {
	Project       string `env:"PGGAT_GC_PROJECT"`
	IpAddressType string `env:"PGGAT_GC_IP_ADDR_TYPE" default:"PRIMARY"`
	AuthUser      string `env:"PGGAT_GC_AUTH_USER" default:"pggat"`
	AuthPassword  string `env:"PGGAT_GC_AUTH_PASSWORD"`
}

func Load() (*Config, error) {
	var conf Config
	gun.Load(&conf)
	if conf.Project == "" {
		return &Config{}, errors.New("expected google cloud project id")
	}
	return &conf, nil
}

func (T *Config) ListenAndServe() error {
	pools, err := NewPools(T)
	if err != nil {
		return err
	}

	go func() {
		var m metrics.Pools
		for {
			m.Clear()
			time.Sleep(1 * time.Minute)
			pools.ReadMetrics(&m)
			log.Print(m.String())
		}
	}()

	var b flip.Bank

	b.Queue(func() error {
		log.Print("listening on :5432")
		return gat.ListenAndServe("tcp", ":5432", frontends.AcceptOptions{
			// TODO(garet) ssl config
			AllowedStartupOptions: []strutil.CIString{
				strutil.MakeCIString("client_encoding"),
				strutil.MakeCIString("datestyle"),
				strutil.MakeCIString("timezone"),
				strutil.MakeCIString("standard_conforming_strings"),
				strutil.MakeCIString("application_name"),
				strutil.MakeCIString("extra_float_digits"),
				strutil.MakeCIString("options"),
			},
		}, gat.NewKeyedPools(pools))
	})

	return b.Wait()
}
