package race

import (
	"reflect"

	"gfx.cafe/gfx/pggat/lib/util/iter"
)

func Recv[T any](next iter.Iter[<-chan T]) (T, bool) {
	cases := casePool.Get()[:0]
	defer func() {
		casePool.Put(cases)
	}()
	iter.ForEach(next, func(ch <-chan T) {
		cases = append(cases, reflect.SelectCase{
			Dir:  reflect.SelectRecv,
			Chan: reflect.ValueOf(ch),
		})
	})
	_, value, ok := reflect.Select(cases)
	if !ok {
		return *new(T), false
	}
	return value.Interface().(T), true
}
