package maths

func Min[T Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}
