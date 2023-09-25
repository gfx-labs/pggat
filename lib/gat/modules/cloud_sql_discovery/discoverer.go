package cloud_sql_discovery

import (
	"context"
	"crypto/tls"
	"net"
	"strings"

	sqladmin "google.golang.org/api/sqladmin/v1beta4"

	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/bouncer/bouncers/v2"
	"pggat/lib/fed"
	"pggat/lib/gat/modules/discovery"
	"pggat/lib/gsql"
)

type authQueryResult struct {
	Username string `sql:"0"`
	Password string `sql:"1"`
}

type Discoverer struct {
	config Config

	google *sqladmin.Service
}

func NewDiscoverer(config Config) (*Discoverer, error) {
	google, err := sqladmin.NewService(context.Background())
	if err != nil {
		return nil, err
	}

	return &Discoverer{
		config: config,
		google: google,
	}, nil
}

func (T *Discoverer) instanceToCluster(primary *sqladmin.DatabaseInstance, replicas ...*sqladmin.DatabaseInstance) (discovery.Cluster, error) {
	var primaryAddress string
	for _, ip := range primary.IpAddresses {
		if ip.Type != T.config.IpAddressType {
			continue
		}
		primaryAddress = net.JoinHostPort(ip.IpAddress, "5432")
	}

	c := discovery.Cluster{
		ID: primary.Name,
		Primary: discovery.Endpoint{
			Network: "tcp",
			Address: primaryAddress,
		},
		Replicas: make(map[string]discovery.Endpoint, len(replicas)),
	}

	for _, replica := range replicas {
		var replicaAddress string
		for _, ip := range primary.IpAddresses {
			if ip.Type != T.config.IpAddressType {
				continue
			}
			replicaAddress = net.JoinHostPort(ip.IpAddress, "5432")
		}
		c.Replicas[replica.Name] = discovery.Endpoint{
			Network: "tcp",
			Address: replicaAddress,
		}
	}

	databases, err := T.google.Databases.List(T.config.Project, primary.Name).Do()
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

	var admin fed.Conn
	defer func() {
		if admin != nil {
			_ = admin.Close()
		}
	}()

	users, err := T.google.Users.List(T.config.Project, primary.Name).Do()
	if err != nil {
		return discovery.Cluster{}, err
	}
	c.Users = make([]discovery.User, 0, len(users.Items))
	for _, user := range users.Items {
		var password string
		if user.Name == T.config.AuthUser {
			password = T.config.AuthPassword
		} else {
			// dial admin connection
			if admin == nil {
				raw, err := net.Dial("tcp", primaryAddress)
				if err != nil {
					return discovery.Cluster{}, err
				}
				admin = fed.WrapNetConn(raw)
				_, err = backends.Accept(&backends.AcceptContext{
					Conn: admin,
					Options: backends.AcceptOptions{
						SSLMode: bouncer.SSLModePrefer,
						SSLConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
						Username:    T.config.AuthUser,
						Credentials: credentials.FromString(T.config.AuthUser, T.config.AuthPassword),
						Database:    c.Databases[0],
					},
				})
				if err != nil {
					return discovery.Cluster{}, err
				}
			}

			var result authQueryResult
			client := new(gsql.Client)
			err := gsql.ExtendedQuery(client, &result, "SELECT usename, passwd FROM pg_shadow WHERE usename=$1", user.Name)
			if err != nil {
				return discovery.Cluster{}, err
			}
			err = client.Close()
			if err != nil {
				return discovery.Cluster{}, err
			}

			initialPacket, err := client.ReadPacket(true, nil)
			if err != nil {
				return discovery.Cluster{}, err
			}
			_, err, err2 := bouncers.Bounce(client, admin, initialPacket)
			if err != nil {
				return discovery.Cluster{}, err
			}
			if err2 != nil {
				return discovery.Cluster{}, err2
			}

			password = result.Password
		}

		c.Users = append(c.Users, discovery.User{
			Username: user.Name,
			Password: password,
		})
	}

	return c, nil
}

func (T *Discoverer) Clusters() ([]discovery.Cluster, error) {
	clusters, err := T.google.Instances.List(T.config.Project).Do()
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

func (T *Discoverer) Updated() <-chan discovery.Cluster {
	return nil
}

func (T *Discoverer) Removed() <-chan string {
	return nil
}

var _ discovery.Discoverer = (*Discoverer)(nil)
