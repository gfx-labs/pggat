package discovery

type Discoverer interface {
	Clusters() ([]Cluster, error)

	Added() <-chan Cluster
	Updated() <-chan Cluster
	Removed() <-chan string
}
