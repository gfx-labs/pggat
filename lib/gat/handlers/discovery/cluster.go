package discovery

type User struct {
	Username string
	Password string
}

type Cluster struct {
	ID string

	Primary  string
	Replicas map[string]string

	Databases []string
	Users     []User
}
