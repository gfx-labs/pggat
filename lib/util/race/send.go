package race

import (
	"reflect"

	"gfx.cafe/gfx/pggat/lib/util/iter"
)

func Send[T any](next iter.Iter[chan<- T], value T) {
	reflectValue := reflect.ValueOf(value)
	cases := casePool.Get()[:0]
	defer func() {
		casePool.Put(cases)
	}()
	iter.ForEach(next, func(ch chan<- T) {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectSend,
			Chan: reflect.ValueOf(ch),
			Send: reflectValue,
		})
	})
	reflect.Select(cases)
}
