package maths

func Clamp[T Ordered](x, min, max T) T {
	return Min(Max(x, min), max)
}
