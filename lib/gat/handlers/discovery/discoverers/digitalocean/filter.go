package digitalocean

import "github.com/digitalocean/godo"

type Filter interface {
	Allow(database godo.Database) bool
	AllowReplica(database godo.DatabaseReplica) bool
}
