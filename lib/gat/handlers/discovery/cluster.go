package discovery

type Endpoint struct {
	Network string
	Address string
}

type User struct {
	Username string
	Password string
}

type Cluster struct {
	ID string

	Primary  Endpoint
	Replicas map[string]Endpoint

	Databases []string
	Users     []User
}
