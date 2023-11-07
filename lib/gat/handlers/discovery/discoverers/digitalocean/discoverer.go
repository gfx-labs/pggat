package digitalocean

import (
	"context"
	"net"
	"strconv"

	"github.com/caddyserver/caddy/v2"
	"github.com/digitalocean/godo"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/discovery"
)

func init() {
	caddy.RegisterModule((*Discoverer)(nil))
}

type Discoverer struct {
	Config

	do *godo.Client
}

func (T *Discoverer) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.handlers.discovery.discoverers.digitalocean",
		New: func() caddy.Module {
			return new(Discoverer)
		},
	}
}

func (T *Discoverer) Provision(ctx caddy.Context) error {
	T.do = godo.NewFromToken(T.APIKey)
	return nil
}

func (T *Discoverer) filter(tags []string) bool {
	for _, tag := range tags {
		if T.Filter.Matches(tag) {
			return true
		}
	}
	return T.Filter.Matches("")
}

func (T *Discoverer) priority(tags []string) int {
	priority := 0
	for _, setter := range T.Priority {
		for _, tag := range tags {
			if setter.Filter.Matches(tag) {
				priority = setter.Value
				break
			}
		}
	}
	return priority
}

func (T *Discoverer) Clusters() ([]discovery.Cluster, error) {
	clusters, _, err := T.do.Databases.List(context.Background(), nil)
	if err != nil {
		return nil, err
	}

	res := make([]discovery.Cluster, 0, len(clusters))
	for _, cluster := range clusters {
		if cluster.EngineSlug != "pg" {
			continue
		}

		// filter by tags
		if !T.filter(cluster.Tags) {
			continue
		}

		var primaryAddr string
		if T.Private {
			primaryAddr = net.JoinHostPort(cluster.PrivateConnection.Host, strconv.Itoa(cluster.PrivateConnection.Port))
		} else {
			primaryAddr = net.JoinHostPort(cluster.Connection.Host, strconv.Itoa(cluster.Connection.Port))
		}

		c := discovery.Cluster{
			ID: cluster.ID,
			Primary: discovery.Node{
				Address:  primaryAddr,
				Priority: T.priority(cluster.Tags),
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

		c.Replicas = make(map[string]discovery.Node, len(replicas))
		for _, replica := range replicas {
			// filter by tags
			if !T.filter(replica.Tags) {
				continue
			}

			var replicaAddr string
			if T.Private {
				replicaAddr = net.JoinHostPort(replica.PrivateConnection.Host, strconv.Itoa(replica.PrivateConnection.Port))
			} else {
				replicaAddr = net.JoinHostPort(replica.Connection.Host, strconv.Itoa(replica.Connection.Port))
			}
			c.Replicas[replica.ID] = discovery.Node{
				Address:  replicaAddr,
				Priority: T.priority(replica.Tags),
			}
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
