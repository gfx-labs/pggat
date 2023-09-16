package digitalocean_discovery

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"time"

	"gfx.cafe/util/go/gun"
	"github.com/digitalocean/godo"
	"tuxpa.in/a/zlog/log"

	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/bouncer/frontends/v0"
	"pggat/lib/gat"
	"pggat/lib/gat/metrics"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/dialer"
	"pggat/lib/gat/pool/pools/session"
	"pggat/lib/gat/pool/pools/transaction"
	"pggat/lib/gat/pool/recipe"
	"pggat/lib/util/flip"
	"pggat/lib/util/strutil"
)

type Config struct {
	APIKey     string `env:"PGGAT_DO_API_KEY"`
	Private    string `env:"PGGAT_DO_PRIVATE"`
	PoolMode   string `env:"PGGAT_POOL_MODE"`
	TLSCrtFile string `env:"PGGAT_TLS_CRT_FILE" default:"/etc/ssl/certs/pgbouncer.crt"`
	TLSKeyFile string `env:"PGGAT_TLS_KEY_FILE" default:"/etc/ssl/certs/pgbouncer.key"`
}

func Load() (Config, error) {
	var conf Config
	gun.Load(&conf)
	if conf.APIKey == "" {
		return Config{}, errors.New("expected auth token")
	}

	return conf, nil
}

func (T *Config) ListenAndServe() error {
	// load certificate
	var sslConfig *tls.Config
	certificate, err := tls.LoadX509KeyPair(T.TLSCrtFile, T.TLSKeyFile)
	if err == nil {
		sslConfig = &tls.Config{
			Certificates: []tls.Certificate{
				certificate,
			},
		}
	} else {
		log.Printf("failed to load certificate, ssl is disabled")
	}

	client := godo.NewFromToken(T.APIKey)
	clusters, _, err := client.Databases.List(context.Background(), nil)

	if err != nil {
		return err
	}

	var pools gat.PoolsMap

	go func() {
		var m metrics.Pools
		for {
			m.Clear()
			time.Sleep(1 * time.Minute)
			pools.ReadMetrics(&m)
			log.Print(m.String())
		}
	}()

	for _, cluster := range clusters {
		if cluster.EngineSlug != "pg" {
			continue
		}

		replicas, _, err := client.Databases.ListReplicas(context.Background(), cluster.ID, nil)
		if err != nil {
			return err
		}

		for _, user := range cluster.Users {
			creds := credentials.Cleartext{
				Username: user.Name,
				Password: user.Password,
			}

			for _, dbname := range cluster.DBNames {
				poolOptions := pool.Options{
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
				}
				if T.PoolMode == "session" {
					poolOptions.ServerResetQuery = "DISCARD ALL"
					poolOptions = session.Apply(poolOptions)
				} else {
					poolOptions = transaction.Apply(poolOptions)
				}

				p := pool.NewPool(poolOptions)

				acceptOptions := backends.AcceptOptions{
					SSLMode: bouncer.SSLModeRequire,
					SSLConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
					Credentials: creds,
					Database:    dbname,
				}

				var addr string
				if T.Private != "" {
					// private
					addr = net.JoinHostPort(cluster.PrivateConnection.Host, strconv.Itoa(cluster.PrivateConnection.Port))
				} else {
					addr = net.JoinHostPort(cluster.Connection.Host, strconv.Itoa(cluster.Connection.Port))
				}

				p.AddRecipe("do", recipe.NewRecipe(recipe.Options{
					Dialer: dialer.Net{
						Network:       "tcp",
						Address:       addr,
						AcceptOptions: acceptOptions,
					},
				}))

				pools.Add(user.Name, dbname, p)
				log.Printf("registered database user=%s database=%s", user.Name, dbname)

				if len(replicas) > 0 {
					// change pool credentials
					creds2 := creds
					creds2.Username = user.Name + "_ro"
					poolOptions2 := poolOptions
					poolOptions2.Credentials = creds2

					p2 := pool.NewPool(poolOptions2)

					for _, replica := range replicas {
						var replicaAddr string
						if T.Private != "" {
							// private
							replicaAddr = net.JoinHostPort(replica.PrivateConnection.Host, strconv.Itoa(replica.PrivateConnection.Port))
						} else {
							replicaAddr = net.JoinHostPort(replica.Connection.Host, strconv.Itoa(replica.Connection.Port))
						}

						p2.AddRecipe("do", recipe.NewRecipe(recipe.Options{
							Dialer: dialer.Net{
								Network:       "tcp",
								Address:       replicaAddr,
								AcceptOptions: acceptOptions,
							},
						}))
					}

					pools.Add(user.Name+"_ro", dbname, p2)
					log.Printf("registered database user=%s database=%s", user.Name+"_ro", dbname)
				}
			}
		}
	}

	var b flip.Bank

	b.Queue(func() error {
		log.Print("listening on :5432")
		return gat.ListenAndServe("tcp", ":5432", frontends.AcceptOptions{
			SSLConfig: sslConfig,
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
