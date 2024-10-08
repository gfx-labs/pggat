package google_cloud_sql

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/util/flip"

	"github.com/caddyserver/caddy/v2"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery"

	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/bouncer/bouncers/v2"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gsql"
)

type authQueryResult struct {
	Username string  `sql:"0"`
	Password *string `sql:"1"`
}

func init() {
	caddy.RegisterModule((*Discoverer)(nil))
}

type Discoverer struct {
	Config

	google *sqladmin.Service
}

func (T *Discoverer) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.discovery.discoverers.google_cloud_sql",
		New: func() caddy.Module {
			return new(Discoverer)
		},
	}
}

func (T *Discoverer) Provision(ctx caddy.Context) error {
	var err error
	T.google, err = sqladmin.NewService(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (T *Discoverer) instanceToCluster(primary *sqladmin.DatabaseInstance, replicas ...*sqladmin.DatabaseInstance) (discovery.Cluster, error) {
	var primaryAddress string
	for _, ip := range primary.IpAddresses {
		if ip.Type != T.IpAddressType {
			continue
		}
		primaryAddress = net.JoinHostPort(ip.IpAddress, "5432")
	}

	c := discovery.Cluster{
		ID: primary.Name,
		Primary: discovery.Node{
			Address: primaryAddress,
		},
		Replicas: make(map[string]discovery.Node, len(replicas)),
	}

	for _, replica := range replicas {
		var replicaAddress string
		for _, ip := range primary.IpAddresses {
			if ip.Type != T.IpAddressType {
				continue
			}
			replicaAddress = net.JoinHostPort(ip.IpAddress, "5432")
		}
		c.Replicas[replica.Name] = discovery.Node{
			Address: replicaAddress,
		}
	}

	databases, err := T.google.Databases.List(T.Project, primary.Name).Do()
	if err != nil {
		return discovery.Cluster{}, err
	}
	c.Databases = make([]string, 0, len(databases.Items))
	for _, database := range databases.Items {
		c.Databases = append(c.Databases, database.Name)
	}

	if len(c.Databases) == 0 {
		return c, nil
	}

	var admin *fed.Conn
	defer func() {
		if admin != nil {
			_ = admin.Close(context.Background())
		}
	}()

	users, err := T.google.Users.List(T.Project, primary.Name).Do()
	if err != nil {
		return discovery.Cluster{}, err
	}
	c.Users = make([]discovery.User, 0, len(users.Items))
	for _, user := range users.Items {
		var password string
		if user.Name == T.AuthUser {
			password = T.AuthPassword
		} else {
			// dial admin connection
			if admin == nil {
				d := pool.Dialer{
					Address: primaryAddress,
					SSLMode: bouncer.SSLModePrefer,
					SSLConfig: &tls.Config{
						InsecureSkipVerify: true,
					},
					Username:    T.AuthUser,
					Credentials: credentials.FromString(T.AuthUser, T.AuthPassword),
					Database:    c.Databases[0],
				}
				admin, err = d.Dial()
				if err != nil {
					return discovery.Cluster{}, err
				}
			}

			var result authQueryResult

			inward, outward, _, _ := gsql.NewPair()

			ctx := context.Background()

			var b flip.Bank
			b.Queue(func() error {
				return gsql.ExtendedQuery(ctx, inward, &result, "SELECT usename, passwd FROM pg_shadow WHERE usename=$1", user.Name)
			})

			b.Queue(func() error {
				initialPacket, err := outward.ReadPacket(ctx, true)
				if err != nil {
					return err
				}
				err, err2 := bouncers.Bounce(ctx, outward, admin, initialPacket)
				if err != nil {
					return err
				}
				if err2 != nil {
					return err2
				}
				return outward.Close(ctx)
			})

			if err = b.Wait(); err != nil {
				return discovery.Cluster{}, err
			}

			if result.Username != user.Name {
				return discovery.Cluster{}, fmt.Errorf(`failed to lookup password for user "%s"`, user.Name)
			}

			if result.Password != nil {
				password = *result.Password
			}
		}

		c.Users = append(c.Users, discovery.User{
			Username: user.Name,
			Password: password,
		})
	}

	return c, nil
}

func (T *Discoverer) Clusters() ([]discovery.Cluster, error) {
	clusters, err := T.google.Instances.List(T.Project).Do()
	if err != nil {
		return nil, err
	}

	res := make([]discovery.Cluster, 0, len(clusters.Items))
	for _, cluster := range clusters.Items {
		if cluster.InstanceType != "CLOUD_SQL_INSTANCE" {
			continue
		}

		if !strings.HasPrefix(cluster.DatabaseVersion, "POSTGRES_") {
			continue
		}

		replicas := make([]*sqladmin.DatabaseInstance, 0, len(cluster.ReplicaNames))
		for _, replicaName := range cluster.ReplicaNames {
			for _, replica := range clusters.Items {
				if replica.Name == replicaName {
					replicas = append(replicas, replica)
					break
				}
			}
		}

		c, err := T.instanceToCluster(cluster, replicas...)
		if err != nil {
			return nil, err
		}
		res = append(res, c)
	}

	return res, nil
}

func (T *Discoverer) Added() <-chan discovery.Cluster {
	return nil
}

func (T *Discoverer) Removed() <-chan string {
	return nil
}

var _ discovery.Discoverer = (*Discoverer)(nil)
var _ caddy.Module = (*Discoverer)(nil)
