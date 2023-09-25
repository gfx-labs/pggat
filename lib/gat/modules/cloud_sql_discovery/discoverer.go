package cloud_sql_discovery

import (
	"context"
	"net"
	"strings"

	sqladmin "google.golang.org/api/sqladmin/v1beta4"

	"pggat/lib/gat/modules/discovery"
)

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

func (T *Discoverer) instanceToCluster(instance *sqladmin.DatabaseInstance) (discovery.Cluster, error) {
	var address string
	for _, ip := range instance.IpAddresses {
		if ip.Type != T.config.IpAddressType {
			continue
		}
		address = net.JoinHostPort(ip.IpAddress, "5432")
	}

	c := discovery.Cluster{
		ID: instance.Name,
		Primary: discovery.Endpoint{
			Network: "tcp",
			Address: address,
		},
	}

	users, err := T.google.Users.List(T.config.Project, instance.Name).Do()
	if err != nil {
		return discovery.Cluster{}, err
	}
	c.Users = make([]discovery.User, 0, len(users.Items))
	for _, user := range users.Items {
		var password string
		if user.Name == T.config.AuthUser {
			password = T.config.AuthPassword
		} else {
			// TODO(garet) lookup password
		}

		c.Users = append(c.Users, discovery.User{
			Username: user.Name,
			Password: password,
		})
	}

	databases, err := T.google.Databases.List(T.config.Project, instance.Name).Do()
	if err != nil {
		return discovery.Cluster{}, err
	}
	c.Databases = make([]string, 0, len(databases.Items))
	for _, database := range databases.Items {
		c.Databases = append(c.Databases, database.Name)
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
		if !strings.HasPrefix(cluster.DatabaseVersion, "POSTGRES_") {
			continue
		}

		c, err := T.instanceToCluster(cluster)
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
