package berr

type Error interface {
	error
	Source() Source
}
