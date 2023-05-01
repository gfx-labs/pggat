package schedulers

type stealer interface {
	steal(ignore *Sink) *Source
}
