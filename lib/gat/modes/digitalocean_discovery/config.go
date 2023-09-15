package digitalocean_discovery

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"gfx.cafe/util/go/gun"
	"github.com/google/uuid"
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
	APIKey   string `env:"PGGAT_DO_API_KEY"`
	PoolMode string `env:"PGGAT_POOL_MODE"`
}

func Load() (Config, error) {
	var conf Config
	gun.Load(&conf)
	if conf.APIKey == "" {
		return Config{}, errors.New("expected auth token")
	}

	return conf, nil
}

func (T *Config) do(endpoint string, resp any) error {
	dest, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	req := http.Request{
		Method: http.MethodGet,
		URL:    dest,
		Header: http.Header{
			"Content-Type":  []string{"application/json"},
			"Authorization": []string{"Bearer " + T.APIKey},
		},
	}

	res, err := http.DefaultClient.Do(&req)
	if err != nil {
		return err
	}

	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		return err
	}

	return nil
}

func (T *Config) ListClusters() ([]Database, error) {
	var res ListClustersResponse
	if err := T.do("https://api.digitalocean.com/v2/databases", &res); err != nil {
		return nil, err
	}
	return res.Databases, nil
}

func (T *Config) ListReplicas(cluster uuid.UUID) ([]Database, error) {
	var res ListReplicasResponse
	if err := T.do(fmt.Sprintf("https://api.digitalocean.com/v2/databases/%s/replicas", cluster.String()), &res); err != nil {
		return nil, err
	}
	return res.Replicas, nil
}

func (T *Config) ListenAndServe() error {
	clusters, err := T.ListClusters()
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
		if cluster.Engine != "pg" {
			continue
		}

		replicas, err := T.ListReplicas(cluster.ID)
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

				p.AddRecipe("do", recipe.NewRecipe(recipe.Options{
					Dialer: dialer.Net{
						Network:       "tcp",
						Address:       net.JoinHostPort(cluster.Connection.Host, strconv.Itoa(cluster.Connection.Port)),
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
						p2.AddRecipe("do", recipe.NewRecipe(recipe.Options{
							Dialer: dialer.Net{
								Network:       "tcp",
								Address:       net.JoinHostPort(replica.Connection.Host, strconv.Itoa(replica.Connection.Port)),
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
