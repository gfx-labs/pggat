package parser

type Builder[O any] func(*Context) (O, bool)
