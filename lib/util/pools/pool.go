package pools

type Pool[T any] []T

func (P *Pool[T]) Get() (T, bool) {
	if len(*P) == 0 {
		return *new(T), false
	}
	v := (*P)[len(*P)-1]
	*P = (*P)[:len(*P)-1]
	return v, true
}

func (P *Pool[T]) Put(v T) {
	*P = append(*P, v)
}
