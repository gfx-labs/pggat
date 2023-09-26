package chans

import (
	"reflect"

	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type MultiRecv[T any] struct {
	cases []reflect.SelectCase
}

func NewMultiRecv[T any](cases []<-chan T) *MultiRecv[T] {
	c := make([]reflect.SelectCase, 0, len(cases))
	for _, ch := range cases {
		c = append(c, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
	}
	return &MultiRecv[T]{
		cases: c,
	}
}

func (c *MultiRecv[T]) Recv() (T, bool) {
	for {
		if len(c.cases) == 0 {
			return *new(T), false
		}

		idx, value, ok := reflect.Select(c.cases)
		if !ok {
			c.cases = slices.DeleteIndex(c.cases, idx)
			continue
		}
		return value.Interface().(T), true
	}
}
