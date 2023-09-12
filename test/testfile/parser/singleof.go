package parser

func SingleOf(ctx *Context, fn func(rune) bool) (rune, bool) {
	c, ok := Any(ctx)
	if !ok {
		return 0, false
	}
	if !fn(c) {
		return 0, false
	}
	return c, true
}

func Single(ctx *Context, r rune) (struct{}, bool) {
	c, ok := Any(ctx)
	if !ok {
		return struct{}{}, false
	}
	if c != r {
		return struct{}{}, false
	}
	return struct{}{}, true
}
