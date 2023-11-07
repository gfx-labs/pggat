package discovery

type User struct {
	Username string
	Password string
}

type Node struct {
	Address  string
	Priority int
}

type Cluster struct {
	ID string

	Primary  Node
	Replicas map[string]Node

	Databases []string
	Users     []User
}
