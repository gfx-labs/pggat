package digitalocean_discovery

import (
	"context"
	"net"
	"strconv"

	"github.com/digitalocean/godo"

	"gfx.cafe/gfx/pggat/lib/gat/modules/discovery"
)

type Discoverer struct {
	config Config

	do *godo.Client
}

func NewDiscoverer(config Config) (*Discoverer, error) {
	return &Discoverer{
		config: config,
		do:     godo.NewFromToken(config.APIKey),
	}, nil
}

func (T Discoverer) Clusters() ([]discovery.Cluster, error) {
	clusters, _, err := T.do.Databases.List(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	res := make([]discovery.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		if cluster.EngineSlug != "pg" {
			continue
		}

		var primaryAddr string
		if T.config.Private {
			primaryAddr = net.JoinHostPort(cluster.PrivateConnection.Host, strconv.Itoa(cluster.PrivateConnection.Port))
		} else {
			primaryAddr = net.JoinHostPort(cluster.Connection.Host, strconv.Itoa(cluster.Connection.Port))
		}

		c := discovery.Cluster{
			ID: cluster.ID,
			Primary: discovery.Endpoint{
				Network: "tcp",
				Address: primaryAddr,
			},
			Databases: cluster.DBNames,
			Users:     make([]discovery.User, 0, len(cluster.Users)),
		}

		for _, user := range cluster.Users {
			c.Users = append(c.Users, discovery.User{
				Username: user.Name,
				Password: user.Password,
			})
		}

		replicas, _, err := T.do.Databases.ListReplicas(context.Background(), cluster.ID, nil)
		if err != nil {
			return nil, err
		}

		c.Replicas = make(map[string]discovery.Endpoint, len(replicas))
		for _, replica := range replicas {
			var replicaAddr string
			if T.config.Private {
				replicaAddr = net.JoinHostPort(replica.PrivateConnection.Host, strconv.Itoa(replica.PrivateConnection.Port))
			} else {
				replicaAddr = net.JoinHostPort(replica.Connection.Host, strconv.Itoa(replica.Connection.Port))
			}
			c.Replicas[replica.ID] = discovery.Endpoint{
				Network: "tcp",
				Address: replicaAddr,
			}
		}

		res = append(res, c)
	}

	return res, nil
}

func (T Discoverer) Added() <-chan discovery.Cluster {
	return nil
}

func (T Discoverer) Updated() <-chan discovery.Cluster {
	return nil
}

func (T Discoverer) Removed() <-chan string {
	return nil
}

var _ discovery.Discoverer = (*Discoverer)(nil)
