package discovery

// Discoverer looks up and returns the servers. It must implement Clusters. Optionally, it can implement Added
// and Removed for faster updating. For updates, just send to Added.
type Discoverer interface {
	Clusters() ([]Cluster, error)

	Added() <-chan Cluster
	Removed() <-chan string
}
