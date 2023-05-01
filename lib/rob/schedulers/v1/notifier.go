package schedulers

type notifier interface {
	notify(which *Source)
}
