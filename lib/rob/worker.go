package rob

type Worker interface {
	Do(ctx *Context, work any)
}
