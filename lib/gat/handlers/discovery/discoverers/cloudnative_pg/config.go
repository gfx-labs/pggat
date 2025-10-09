package cloudnative_pg

type Config struct {
	// Namespace to watch for CloudNativePG clusters
	// If empty, watches all namespaces
	Namespace string `json:"namespace,omitempty"`

	// ClusterDomain is the Kubernetes cluster domain (default: cluster.local)
	ClusterDomain string `json:"cluster_domain,omitempty"`

	// ServiceSuffix for different service types
	// CloudNativePG creates multiple services:
	// - <cluster-name>-rw (read-write service pointing to primary)
	// - <cluster-name>-r (read service for load balancing across all instances)
	// - <cluster-name>-ro (read-only service for replicas only)
	ReadWriteServiceSuffix string `json:"read_write_service_suffix,omitempty"`
	ReadOnlyServiceSuffix  string `json:"read_only_service_suffix,omitempty"`

	// Port is the PostgreSQL port (default: 5432)
	Port int `json:"port,omitempty"`

	// SecretSuffix for app user secrets
	// CloudNativePG creates secrets like <cluster-name>-app by default
	SecretSuffix string `json:"secret_suffix,omitempty"`

	// IncludeSuperuser includes the postgres superuser in discovered users
	IncludeSuperuser bool `json:"include_superuser,omitempty"`
}
