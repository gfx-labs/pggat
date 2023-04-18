package race

import "reflect"

func Send[T any](next func(int) (chan<- T, bool), value T) bool {
	reflectValue := reflect.ValueOf(value)
	cases := casePool.Get()[:0]
	defer func() {
		casePool.Put(cases)
	}()
	for i := 0; ; i++ {
		if ch, ok := next(i); ok {
			cases = append(cases, reflect.SelectCase{
				Dir:  reflect.SelectSend,
				Chan: reflect.ValueOf(ch),
				Send: reflectValue,
			})
		} else {
			break
		}
	}
	_, _, ok := reflect.Select(cases)
	return ok
}
