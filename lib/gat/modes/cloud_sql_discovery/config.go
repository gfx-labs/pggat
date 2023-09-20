package cloud_sql_discovery

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strings"
	"time"

	"gfx.cafe/util/go/gun"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
	"tuxpa.in/a/zlog/log"

	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/bouncer/frontends/v0"
	"pggat/lib/gat"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/dialer"
	"pggat/lib/gat/pool/pools/transaction"
	"pggat/lib/gat/pool/recipe"
	"pggat/lib/util/flip"
	"pggat/lib/util/strutil"
)

type Config struct {
	Project       string `env:"PGGAT_GC_PROJECT"`
	IpAddressType string `env:"PGGAT_GC_IP_ADDR_TYPE" default:"PRIMARY"`
}

func Load() (Config, error) {
	var conf Config
	gun.Load(&conf)
	if conf.Project == "" {
		return Config{}, errors.New("expected google cloud project id")
	}
	return conf, nil
}

func (T *Config) ListenAndServe() error {
	service, err := sqladmin.NewService(context.Background())
	if err != nil {
		return err
	}

	instances, err := service.Instances.List(T.Project).Do()
	if err != nil {
		return err
	}

	var pools gat.PoolsMap

	for _, instance := range instances.Items {
		if !strings.HasPrefix(instance.DatabaseVersion, "POSTGRES_") {
			continue
		}

		var address string
		for _, ip := range instance.IpAddresses {
			if ip.Type != T.IpAddressType {
				continue
			}
			address = net.JoinHostPort(ip.IpAddress, "5432")
		}
		if address == "" {
			continue
		}

		users, err := service.Users.List(T.Project, instance.Name).Do()
		if err != nil {
			return err
		}
		databases, err := service.Databases.List(T.Project, instance.Name).Do()
		if err != nil {
			return err
		}
		for _, user := range users.Items {
			creds := credentials.Cleartext{
				Username: user.Name,
				Password: "password",
			}

			for _, database := range databases.Items {
				d := dialer.Net{
					Network: "tcp",
					Address: address,
					AcceptOptions: backends.AcceptOptions{
						SSLMode: bouncer.SSLModePrefer,
						SSLConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
						Username:    user.Name,
						Credentials: creds,
						Database:    database.Name,
					},
				}

				options := transaction.Apply(pool.Options{
					Credentials:                creds,
					ServerReconnectInitialTime: 5 * time.Second,
					ServerReconnectMaxTime:     5 * time.Second,
					ServerIdleTimeout:          5 * time.Minute,
					TrackedParameters: []strutil.CIString{
						strutil.MakeCIString("client_encoding"),
						strutil.MakeCIString("datestyle"),
						strutil.MakeCIString("timezone"),
						strutil.MakeCIString("standard_conforming_strings"),
						strutil.MakeCIString("application_name"),
					},
				})

				p := pool.NewPool(options)
				p.AddRecipe(instance.Name, recipe.NewRecipe(recipe.Options{
					Dialer: d,
				}))

				pools.Add(user.Name, database.Name, p)
				log.Printf("registered database user=%s database=%s", user.Name, database.Name)
			}
		}
	}

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
		}, &pools)
	})

	return b.Wait()
}
