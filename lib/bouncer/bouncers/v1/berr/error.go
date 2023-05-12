package berr

type Error interface {
	IsServer() bool
	IsClient() bool

	err()
}
