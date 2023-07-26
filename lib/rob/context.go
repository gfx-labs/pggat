package rob

type Context struct {
	Constraints Constraints
	Removed     bool
}

func (T *Context) Remove() {
	T.Removed = true
}

func (T *Context) Reset() {
	T.Removed = false
}
