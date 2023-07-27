package rob

type Context struct {
	OnWait      chan<- struct{}
	Constraints Constraints
	Removed     bool
}

func (T *Context) Remove() {
	T.Removed = true
}

func (T *Context) Reset() {
	T.Removed = false
}
