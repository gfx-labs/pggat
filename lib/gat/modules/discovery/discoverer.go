package discovery

// Discoverer looks up and returns the servers. It must implement either Clusters or Added, Updated, and Removed.
// Both can be implemented for extra robustness.
type Discoverer interface {
	Clusters() ([]Cluster, error)

	Added() <-chan Cluster
	Updated() <-chan Cluster
	Removed() <-chan string
}
