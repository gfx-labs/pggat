package digitalocean_discovery

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"gfx.cafe/util/go/gun"
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

	var m gat.PoolsMap

	for _, cluster := range r.Databases {
		if cluster.Engine != "pg" {
			continue
		}

		for _, user := range cluster.Users {
			creds := credentials.Cleartext{
				Username: user.Name,
				Password: user.Password,
			}

			for _, dbname := range cluster.DBNames {
				p := pool.NewPool(transaction.Apply(pool.Options{
					Credentials: creds,
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

				m.Add(user.Name, dbname, p)
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
		}, &m)
	})

	return b.Wait()
}
