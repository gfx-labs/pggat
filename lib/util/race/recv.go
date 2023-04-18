package race

import "reflect"

func Recv[T any](next func(int) (<-chan T, bool)) (T, bool) {
	cases := casePool.Get()[:0]
	defer func() {
		casePool.Put(cases)
	}()
	for i := 0; ; i++ {
		if ch, ok := next(i); ok {
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectRecv,
				Chan: reflect.ValueOf(ch),
			})
		} else {
			break
		}
	}
	_, value, ok := reflect.Select(cases)
	if !ok {
		return *new(T), false
	}
	return value.Interface().(T), true
}
