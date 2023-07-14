package berr

type Error interface {
	error
	err()
}
