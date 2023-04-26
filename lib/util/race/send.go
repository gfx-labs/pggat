package race

import (
	"reflect"

	"gfx.cafe/gfx/pggat/lib/util/iter"
)

func Send[T any](next iter.Iter[chan<- T], value T) bool {
	reflectValue := reflect.ValueOf(value)
	cases := casePool.Get()[:0]
	defer func() {
		casePool.Put(cases)
	}()
	for ch, ok := next(); ok; ch, ok = next() {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectSend,
			Chan: reflect.ValueOf(ch),
			Send: reflectValue,
		})
	}
	_, _, ok := reflect.Select(cases)
	return ok
}
