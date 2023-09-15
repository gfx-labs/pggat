package digitalocean_discovery

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"gfx.cafe/util/go/gun"
	"tuxpa.in/a/zlog/log"

	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/bouncer/frontends/v0"
	"pggat/lib/gat"
	"pggat/lib/gat/metrics"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/dialer"
	"pggat/lib/gat/pool/pools/transaction"
	"pggat/lib/gat/pool/recipe"
	"pggat/lib/util/flip"
	"pggat/lib/util/strutil"
)

type Config struct {
	APIKey string `env:"PGGAT_DO_API_KEY"`
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
	dest, err := url.Parse("https://api.digitalocean.com/v2/databases")
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

	resp, err := http.DefaultClient.Do(&req)
	if err != nil {
		return err
	}

	var r ListClustersResponse
	err = json.NewDecoder(resp.Body).Decode(&r)
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

	for _, cluster := range r.Databases {
		if cluster.Engine != "pg" {
			continue
		}

		replicaDest, err := url.Parse("https://api.digitalocean.com/v2/databases/" + cluster.ID.String() + "/replicas")
		if err != nil {
			return err
		}

		replicaReq := http.Request{
			Method: http.MethodGet,
			URL:    replicaDest,
			Header: http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{"Bearer " + T.APIKey},
			},
		}

		replicaResp, err := http.DefaultClient.Do(&replicaReq)
		if err != nil {
			return err
		}

		var replicaR ListReplicasResponse
		err = json.NewDecoder(replicaResp.Body).Decode(&replicaR)
		if err != nil {
			return err
		}

		for _, user := range cluster.Users {
			creds := credentials.Cleartext{
				Username: user.Name,
				Password: user.Password,
			}

			for _, dbname := range cluster.DBNames {
				p := pool.NewPool(transaction.Apply(pool.Options{
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
				}))
				p.AddRecipe("do", recipe.NewRecipe(recipe.Options{
					Dialer: dialer.Net{
						Network: "tcp",
						Address: net.JoinHostPort(cluster.Connection.Host, strconv.Itoa(cluster.Connection.Port)),
						AcceptOptions: backends.AcceptOptions{
							SSLMode: bouncer.SSLModeRequire,
							SSLConfig: &tls.Config{
								InsecureSkipVerify: true,
							},
							Credentials: creds,
							Database:    dbname,
						},
					},
				}))

				pools.Add(user.Name, dbname, p)

				if len(replicaR.Replicas) > 0 {
					creds2 := creds
					creds2.Username = user.Name + "_ro"
					p2 := pool.NewPool(transaction.Apply(pool.Options{
						Credentials:                creds2,
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
					}))

					for _, replica := range replicaR.Replicas {
						p2.AddRecipe("do", recipe.NewRecipe(recipe.Options{
							Dialer: dialer.Net{
								Network: "tcp",
								Address: net.JoinHostPort(replica.Connection.Host, strconv.Itoa(replica.Connection.Port)),
								AcceptOptions: backends.AcceptOptions{
									SSLMode: bouncer.SSLModeRequire,
									SSLConfig: &tls.Config{
										InsecureSkipVerify: true,
									},
									Credentials: creds,
									Database:    dbname,
								},
							},
						}))
					}

					pools.Add(user.Name+"_ro", dbname, p2)
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
