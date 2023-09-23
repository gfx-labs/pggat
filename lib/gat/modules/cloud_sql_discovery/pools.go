package cloud_sql_discovery

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"strings"
	"time"

	sqladmin "google.golang.org/api/sqladmin/v1beta4"
	"tuxpa.in/a/zlog/log"

	"pggat/lib/auth"
	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/gat"
	"pggat/lib/gat/metrics"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/pools/transaction"
	"pggat/lib/gat/pool/recipe"
	"pggat/lib/gsql"
	"pggat/lib/util/maps"
	"pggat/lib/util/strutil"
)

type authQueryResult struct {
	Username string `sql:"0"`
	Password string `sql:"1"`
}

type poolKey struct {
	User     string
	Database string
}

type poolTemplate struct {
	Address string
}

type Pools struct {
	Config *Config

	templates maps.RWLocked[poolKey, poolTemplate]
	pools     maps.RWLocked[poolKey, *pool.Pool]
}

func NewPools(config *Config) (*Pools, error) {
	p := &Pools{
		Config: config,
	}

	if err := p.init(); err != nil {
		return nil, err
	}

	return p, nil
}

func (T *Pools) init() error {
	service, err := sqladmin.NewService(context.Background())
	if err != nil {
		return err
	}

	instances, err := service.Instances.List(T.Config.Project).Do()
	if err != nil {
		return err
	}

	for _, instance := range instances.Items {
		if !strings.HasPrefix(instance.DatabaseVersion, "POSTGRES_") {
			continue
		}

		var address string
		for _, ip := range instance.IpAddresses {
			if ip.Type != T.Config.IpAddressType {
				continue
			}
			address = net.JoinHostPort(ip.IpAddress, "5432")
		}
		if address == "" {
			continue
		}

		users, err := service.Users.List(T.Config.Project, instance.Name).Do()
		if err != nil {
			return err
		}
		databases, err := service.Databases.List(T.Config.Project, instance.Name).Do()
		if err != nil {
			return err
		}
		for _, user := range users.Items {
			for _, database := range databases.Items {
				T.templates.Store(poolKey{
					User:     user.Name,
					Database: database.Name,
				}, poolTemplate{
					Address: address,
				})
				log.Printf("registered database user=%s database=%s", user.Name, database.Name)
			}
		}
	}

	return nil
}

func (T *Pools) Lookup(user, database string) *pool.Pool {
	p, ok := T.pools.Load(poolKey{
		User:     user,
		Database: database,
	})
	if ok {
		return p
	}
	template, ok := T.templates.Load(poolKey{
		User:     user,
		Database: database,
	})
	if !ok {
		return nil
	}

	var creds auth.Credentials
	if user == T.Config.AuthUser {
		creds = credentials.Cleartext{
			Username: user,
			Password: T.Config.AuthPassword,
		}
	} else {
		// query for password
		authPool := T.Lookup(T.Config.AuthUser, database)
		if authPool == nil {
			return nil
		}

		var result authQueryResult
		client := new(gsql.Client)
		err := gsql.ExtendedQuery(client, &result, "SELECT usename, passwd FROM pg_shadow WHERE usename=$1", user)
		if err != nil {
			log.Println("auth query failed:", err)
			return nil
		}
		err = client.Close()
		if err != nil {
			log.Println("auth query failed:", err)
			return nil
		}
		err = authPool.ServeBot(client)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Println("auth query failed:", err)
			return nil
		}

		if result.Username != user {
			// user not found
			return nil
		}

		creds = credentials.FromString(result.Username, result.Password)
	}

	d := recipe.Dialer{
		Network: "tcp",
		Address: template.Address,
		AcceptOptions: backends.AcceptOptions{
			SSLMode: bouncer.SSLModePrefer,
			SSLConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			Username:    user,
			Credentials: creds,
			Database:    database,
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

	p = pool.NewPool(options)
	p.AddRecipe("gc", recipe.NewRecipe(recipe.Options{
		Dialer: d,
	}))

	T.pools.Store(poolKey{
		User:     user,
		Database: database,
	}, p)
	return p
}

func (T *Pools) ReadMetrics(metrics *metrics.Pools) {
	T.pools.Range(func(_ poolKey, p *pool.Pool) bool {
		p.ReadMetrics(&metrics.Pool)
		return true
	})
}

var _ gat.Pools = (*Pools)(nil)
